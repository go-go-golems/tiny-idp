// Command auditlint runs repository-specific production-readiness analyzers.
//
// These checks intentionally complement, rather than replace, go vet,
// staticcheck, gosec, and govulncheck. They encode tiny-idp trust-boundary and
// persistence invariants that general-purpose tools cannot infer.
package main

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

func main() {
	multichecker.Main(
		internalAPIAnalyzer,
		ignoredRandAnalyzer,
		httpServerAnalyzer,
		securityDefaultAnalyzer,
		rateLimitKeyAnalyzer,
		unusedConfigAnalyzer,
		auditDeliveryAnalyzer,
		atomicityAnalyzer,
		backupCopyAnalyzer,
	)
}

var internalAPIAnalyzer = &analysis.Analyzer{
	Name: "tinyidpinternalapi",
	Doc:  "reports exported APIs whose types depend on Go internal packages",
	Run:  runInternalAPI,
}

func runInternalAPI(pass *analysis.Pass) (any, error) {
	if !strings.Contains(pass.Pkg.Path(), "/pkg/") {
		return nil, nil
	}
	for _, name := range pass.Pkg.Scope().Names() {
		obj := pass.Pkg.Scope().Lookup(name)
		typeName, ok := obj.(*types.TypeName)
		if !ok || !typeName.Exported() || typeName.IsAlias() {
			continue
		}
		paths := map[string]struct{}{}
		collectInternalPaths(obj.Type(), map[types.Type]bool{}, paths)
		if len(paths) == 0 {
			continue
		}
		values := make([]string, 0, len(paths))
		for path := range paths {
			values = append(values, path)
		}
		sort.Strings(values)
		pass.Reportf(obj.Pos(), "exported type %q depends on internal package(s): %s; external modules cannot implement or construct this API", obj.Name(), strings.Join(values, ", "))
	}
	return nil, nil
}

func collectInternalPaths(t types.Type, seen map[types.Type]bool, out map[string]struct{}) {
	if t == nil || seen[t] {
		return
	}
	seen[t] = true
	t = types.Unalias(t)
	switch value := t.(type) {
	case *types.Named:
		if obj := value.Obj(); obj != nil && obj.Pkg() != nil {
			path := obj.Pkg().Path()
			if strings.Contains(path, "/internal/") {
				out[path] = struct{}{}
				return
			}
		}
		collectInternalPaths(value.Underlying(), seen, out)
		for i := 0; i < value.TypeArgs().Len(); i++ {
			collectInternalPaths(value.TypeArgs().At(i), seen, out)
		}
	case *types.Pointer:
		collectInternalPaths(value.Elem(), seen, out)
	case *types.Slice:
		collectInternalPaths(value.Elem(), seen, out)
	case *types.Array:
		collectInternalPaths(value.Elem(), seen, out)
	case *types.Map:
		collectInternalPaths(value.Key(), seen, out)
		collectInternalPaths(value.Elem(), seen, out)
	case *types.Chan:
		collectInternalPaths(value.Elem(), seen, out)
	case *types.Struct:
		for i := 0; i < value.NumFields(); i++ {
			collectInternalPaths(value.Field(i).Type(), seen, out)
		}
	case *types.Signature:
		collectTupleInternalPaths(value.Params(), seen, out)
		collectTupleInternalPaths(value.Results(), seen, out)
	case *types.Interface:
		for i := 0; i < value.NumExplicitMethods(); i++ {
			collectInternalPaths(value.ExplicitMethod(i).Type(), seen, out)
		}
	}
}

func collectTupleInternalPaths(tuple *types.Tuple, seen map[types.Type]bool, out map[string]struct{}) {
	if tuple == nil {
		return
	}
	for i := 0; i < tuple.Len(); i++ {
		collectInternalPaths(tuple.At(i).Type(), seen, out)
	}
}

var ignoredRandAnalyzer = &analysis.Analyzer{
	Name:     "tinyidprand",
	Doc:      "reports ignored errors from crypto/rand.Read",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runIgnoredRand,
}

func runIgnoredRand(pass *analysis.Pass) (any, error) {
	ins := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	ins.Preorder([]ast.Node{(*ast.AssignStmt)(nil), (*ast.ExprStmt)(nil)}, func(node ast.Node) {
		switch value := node.(type) {
		case *ast.AssignStmt:
			if len(value.Rhs) != 1 || !isCallTo(pass, value.Rhs[0], "crypto/rand", "Read") {
				return
			}
			if len(value.Lhs) >= 2 && isBlank(value.Lhs[1]) {
				pass.Reportf(value.Pos(), "error from crypto/rand.Read is ignored; fail closed when the CSPRNG is unavailable")
			}
		case *ast.ExprStmt:
			if isCallTo(pass, value.X, "crypto/rand", "Read") {
				pass.Reportf(value.Pos(), "results from crypto/rand.Read are ignored; fail closed when the CSPRNG is unavailable")
			}
		}
	})
	return nil, nil
}

var httpServerAnalyzer = &analysis.Analyzer{
	Name:     "tinyidphttpserver",
	Doc:      "reports package-level HTTP serving that cannot configure timeouts or graceful shutdown",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run: func(pass *analysis.Pass) (any, error) {
		inspectCalls(pass, func(call *ast.CallExpr) {
			if isCallTo(pass, call, "net/http", "ListenAndServe") {
				pass.Reportf(call.Pos(), "http.ListenAndServe uses a zero-value Server: construct http.Server with read-header/idle limits and an explicit Shutdown path")
			}
		})
		return nil, nil
	},
}

var securityDefaultAnalyzer = &analysis.Analyzer{
	Name:     "tinyidpsecuritydefault",
	Doc:      "reports silent no-op audit and allow-all rate-limit defaults",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run: func(pass *analysis.Pass) (any, error) {
		ins := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
		ins.Preorder([]ast.Node{(*ast.CompositeLit)(nil)}, func(node ast.Node) {
			lit := node.(*ast.CompositeLit)
			t := pass.TypesInfo.TypeOf(lit)
			named, _ := types.Unalias(t).(*types.Named)
			if named == nil || named.Obj() == nil {
				return
			}
			path := ""
			if named.Obj().Pkg() != nil {
				path = named.Obj().Pkg().Path()
			}
			switch {
			case named.Obj().Name() == "NoopSink" && strings.HasSuffix(path, "/pkg/idp"):
				pass.Reportf(lit.Pos(), "NoopSink silently discards security audit events; production construction should require an explicit durable sink")
			case named.Obj().Name() == "AllowAllRateLimiter" && strings.HasSuffix(path, "/internal/fositeadapter"):
				pass.Reportf(lit.Pos(), "AllowAllRateLimiter silently disables request throttling; production construction should require an explicit limiter")
			}
		})
		return nil, nil
	},
}

var rateLimitKeyAnalyzer = &analysis.Analyzer{
	Name:     "tinyidpratelimitkey",
	Doc:      "reports net/http RemoteAddr values used directly in rate-limit keys",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run: func(pass *analysis.Pass) (any, error) {
		inspectCalls(pass, func(call *ast.CallExpr) {
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Allow" {
				return
			}
			for _, arg := range call.Args {
				if containsSelector(arg, "RemoteAddr") {
					pass.Reportf(arg.Pos(), "rate-limit key includes http.Request.RemoteAddr (IP:port); normalize the trusted client IP or each new connection can receive a fresh bucket")
				}
			}
		})
		return nil, nil
	},
}

var unusedConfigAnalyzer = &analysis.Analyzer{
	Name:     "tinyidpconfiguse",
	Doc:      "reports exported configuration fields that the defining public package never reads",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runUnusedConfig,
}

func runUnusedConfig(pass *analysis.Pass) (any, error) {
	if !strings.Contains(pass.Pkg.Path(), "/pkg/") {
		return nil, nil
	}
	used := map[types.Object]bool{}
	for _, obj := range pass.TypesInfo.Uses {
		used[obj] = true
	}
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			typeSpec, ok := node.(*ast.TypeSpec)
			if !ok || (!strings.HasSuffix(typeSpec.Name.Name, "Config") && !strings.HasSuffix(typeSpec.Name.Name, "Options")) {
				return true
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range structType.Fields.List {
				for _, name := range field.Names {
					obj := pass.TypesInfo.Defs[name]
					if obj != nil && obj.Exported() && !used[obj] {
						pass.Reportf(name.Pos(), "exported configuration field %s.%s is never read by package %s; callers can set it but behavior does not change", typeSpec.Name.Name, name.Name, pass.Pkg.Path())
					}
				}
			}
			return false
		})
	}
	return nil, nil
}

var auditDeliveryAnalyzer = &analysis.Analyzer{
	Name:     "tinyidpauditdelivery",
	Doc:      "reports explicitly ignored errors from audit Sink.Emit",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run: func(pass *analysis.Pass) (any, error) {
		ins := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
		ins.Preorder([]ast.Node{(*ast.AssignStmt)(nil), (*ast.ExprStmt)(nil)}, func(node ast.Node) {
			switch value := node.(type) {
			case *ast.AssignStmt:
				if len(value.Lhs) == 1 && isBlank(value.Lhs[0]) && len(value.Rhs) == 1 && isMethodCall(value.Rhs[0], "Emit") {
					pass.Reportf(value.Pos(), "audit Sink.Emit error is discarded; define delivery/backpressure/failure semantics for production audit evidence")
				}
			case *ast.ExprStmt:
				if isMethodCall(value.X, "Emit") {
					pass.Reportf(value.Pos(), "audit Sink.Emit result is discarded; define delivery/backpressure/failure semantics for production audit evidence")
				}
			}
		})
		return nil, nil
	},
}

var atomicityAnalyzer = &analysis.Analyzer{
	Name:     "tinyidpatomicity",
	Doc:      "reports persistence functions with multiple mutations but no explicit transaction",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runAtomicity,
}

func runAtomicity(pass *analysis.Pass) (any, error) {
	path := pass.Pkg.Path()
	if !strings.Contains(path, "/pkg/sqlitestore") && !strings.Contains(path, "/internal/admin") && !strings.Contains(path, "/internal/fositeadapter") {
		return nil, nil
	}
	for _, file := range pass.Files {
		if strings.HasSuffix(pass.Fset.Position(file.Pos()).Filename, "_test.go") {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Name.Name == "Open" || hasDirective(fn.Doc, "tinyidp:transaction-scoped") {
				continue
			}
			mutations := 0
			mutationInLoop := false
			hasTransaction := false
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				switch value := node.(type) {
				case *ast.CallExpr:
					sel, ok := value.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					if sel.Sel.Name == "Begin" || sel.Sel.Name == "BeginTx" || sel.Sel.Name == "Update" || isAtomicBoundary(sel.Sel.Name) {
						hasTransaction = true
					}
					if isMutationName(sel.Sel.Name) {
						mutations++
					}
				}
				return true
			})
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				var body ast.Node
				switch loop := node.(type) {
				case *ast.ForStmt:
					body = loop.Body
				case *ast.RangeStmt:
					body = loop.Body
				default:
					return true
				}
				ast.Inspect(body, func(child ast.Node) bool {
					call, ok := child.(*ast.CallExpr)
					if !ok {
						return !mutationInLoop
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if ok && isMutationName(sel.Sel.Name) {
						mutationInLoop = true
					}
					return !mutationInLoop
				})
				return !mutationInLoop
			})
			if !hasTransaction && (mutations >= 2 || mutationInLoop) {
				pass.Reportf(fn.Name.Pos(), "persistence function %s performs %d mutation operations without Begin/BeginTx; partial failure or concurrency can expose intermediate state", fn.Name.Name, mutations)
			}
		}
	}
	return nil, nil
}

func isAtomicBoundary(name string) bool {
	switch name {
	case "CreateUserWithCredential", "ReplacePasswordAndSecurityState", "RecordFailedLogin", "RecordSuccessfulLogin", "RotateSigningKey", "RotateRefreshToken", "RevokeRefreshTokenFamily":
		return true
	default:
		return false
	}
}

func hasDirective(group *ast.CommentGroup, directive string) bool {
	return group != nil && strings.Contains(group.Text(), directive)
}

func isMutationName(name string) bool {
	for _, prefix := range []string{"Exec", "Put", "Create", "Rotate", "Revoke", "Activate", "Retire", "Reset", "Delete", "Mark", "put", "revoke"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

var backupCopyAnalyzer = &analysis.Analyzer{
	Name:     "tinyidpbackupcopy",
	Doc:      "reports raw file copying in SQLite backup code",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run: func(pass *analysis.Pass) (any, error) {
		if !strings.Contains(pass.Pkg.Path(), "/internal/admin") {
			return nil, nil
		}
		for _, file := range pass.Files {
			filename := filepath.Base(pass.Fset.Position(file.Pos()).Filename)
			if filename != "backup.go" {
				continue
			}
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if ok && isCallTo(pass, call, "io", "Copy") {
					pass.Reportf(call.Pos(), "raw io.Copy is not a consistent live SQLite backup (especially in WAL mode); use SQLite online backup or VACUUM INTO")
				}
				return true
			})
		}
		return nil, nil
	},
}

func inspectCalls(pass *analysis.Pass, fn func(*ast.CallExpr)) {
	ins := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	ins.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(node ast.Node) { fn(node.(*ast.CallExpr)) })
}

func isCallTo(pass *analysis.Pass, expr ast.Expr, pkgPath, name string) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	var obj types.Object
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		obj = pass.TypesInfo.Uses[fun]
	case *ast.SelectorExpr:
		obj = pass.TypesInfo.Uses[fun.Sel]
	}
	fn, ok := obj.(*types.Func)
	return ok && fn.Pkg() != nil && fn.Pkg().Path() == pkgPath && fn.Name() == name
}

func isBlank(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == "_"
}

func isMethodCall(expr ast.Expr, name string) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == name
}

func containsSelector(expr ast.Expr, name string) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		sel, ok := node.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == name {
			found = true
			return false
		}
		return !found
	})
	return found
}
