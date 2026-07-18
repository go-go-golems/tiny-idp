// Package idpuianalyzer defines static checks for interaction renderer code.
package idpuianalyzer

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "idpuianalyzer",
	Doc:  "checks tiny-idp renderers for unsafe template and HTTP coupling patterns",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}
		imports := importAliases(file)
		for _, spec := range file.Imports {
			pathValue, _ := strconv.Unquote(spec.Path.Value)
			if pathValue == "text/template" {
				pass.Reportf(spec.Pos(), "use html/template, not text/template, for interaction HTML")
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.FuncDecl:
				if typed.Name.Name == "RenderInteraction" && broadHTTPAPI(typed.Type, imports) {
					pass.Reportf(typed.Name.Pos(), "RenderInteraction must not accept http.ResponseWriter or *http.Request")
				}
			case *ast.InterfaceType:
				for _, method := range typed.Methods.List {
					if len(method.Names) == 1 && method.Names[0].Name == "RenderInteraction" {
						if signature, ok := method.Type.(*ast.FuncType); ok && broadHTTPAPI(signature, imports) {
							pass.Reportf(method.Names[0].Pos(), "RenderInteraction must not accept http.ResponseWriter or *http.Request")
						}
					}
				}
			case *ast.CallExpr:
				checkCall(pass, typed, imports)
			}
			return true
		})
	}
	return nil, nil
}

func checkCall(pass *analysis.Pass, call *ast.CallExpr, imports map[string]string) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	packageName, ok := selector.X.(*ast.Ident)
	if !ok {
		return
	}
	importPath := imports[packageName.Name]
	if importPath == "html/template" && trustedContentTypes[selector.Sel.Name] {
		pass.Reportf(call.Pos(), "trusted-content conversion template.%s is forbidden in interaction renderers", selector.Sel.Name)
	}
	if (importPath == "fmt" && strings.HasPrefix(selector.Sel.Name, "Fprint")) || (importPath == "io" && selector.Sel.Name == "WriteString") {
		if callContainsHTMLLiteral(call) {
			pass.Reportf(call.Pos(), "direct HTML string construction is forbidden; use html/template")
		}
	}
	if selector.Sel.Name == "Write" && callContainsHTMLLiteral(call) {
		pass.Reportf(call.Pos(), "direct HTML string construction is forbidden; use html/template")
	}
}

func callContainsHTMLLiteral(call *ast.CallExpr) bool {
	found := false
	for _, arg := range call.Args {
		ast.Inspect(arg, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err == nil && containsHTMLTag(value) {
				found = true
				return false
			}
			return true
		})
	}
	return found
}

func containsHTMLTag(value string) bool {
	lower := strings.ToLower(value)
	for _, fragment := range []string{"<!doctype", "<html", "<form", "<input", "<button", "<script", "<style", "<iframe"} {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

func broadHTTPAPI(signature *ast.FuncType, imports map[string]string) bool {
	if signature == nil || signature.Params == nil {
		return false
	}
	for _, field := range signature.Params.List {
		if isBroadHTTPType(field.Type, imports) {
			return true
		}
	}
	return false
}

func isBroadHTTPType(expression ast.Expr, imports map[string]string) bool {
	switch typed := expression.(type) {
	case *ast.StarExpr:
		return isBroadHTTPType(typed.X, imports)
	case *ast.SelectorExpr:
		packageName, ok := typed.X.(*ast.Ident)
		return ok && imports[packageName.Name] == "net/http" && (typed.Sel.Name == "ResponseWriter" || typed.Sel.Name == "Request")
	case *ast.Ellipsis:
		return isBroadHTTPType(typed.Elt, imports)
	}
	return false
}

func importAliases(file *ast.File) map[string]string {
	result := make(map[string]string, len(file.Imports))
	for _, spec := range file.Imports {
		pathValue, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		name := ""
		if spec.Name != nil {
			name = spec.Name.Name
		} else {
			parts := strings.Split(pathValue, "/")
			name = parts[len(parts)-1]
		}
		result[name] = pathValue
	}
	return result
}

var trustedContentTypes = map[string]bool{
	"CSS": true, "HTML": true, "HTMLAttr": true, "JS": true,
	"JSStr": true, "Srcset": true, "URL": true,
}
