## Documentation

### Overview

Package ssa defines a representation of the elements of Go programs (packages, types, functions, variables and constants) using a static single-assignment (SSA) form intermediate representation (IR) for the bodies of functions.

For an introduction to SSA form, see [http://en.wikipedia.org/wiki/Static\_single\_assignment\_form](http://en.wikipedia.org/wiki/Static_single_assignment_form). This page provides a broader reading list: [http://www.dcs.gla.ac.uk/~jsinger/ssa.html](http://www.dcs.gla.ac.uk/~jsinger/ssa.html).

The level of abstraction of the SSA form is intentionally close to the source language to facilitate construction of source analysis tools. It is not intended for machine code generation.

All looping, branching and switching constructs are replaced with unstructured control flow. Higher-level control flow constructs such as multi-way branch can be reconstructed as needed; see [golang.org/x/tools/go/ssa/ssautil.Switches](https://pkg.go.dev/golang.org/x/tools@v0.48.0/go/ssa/ssautil#Switches) for an example.

The simplest way to create the SSA representation of a package is to load typed syntax trees using [golang.org/x/tools/go/packages](https://pkg.go.dev/golang.org/x/tools@v0.48.0/go/packages), then invoke the [golang.org/x/tools/go/ssa/ssautil.Packages](https://pkg.go.dev/golang.org/x/tools@v0.48.0/go/ssa/ssautil#Packages) helper function. (See the package-level Examples named LoadPackages and LoadWholeProgram.) The resulting [ssa.Program](#Program) contains all the packages and their members, but SSA code is not created for function bodies until a subsequent call to [Package.Build](#Package.Build) or [Program.Build](#Program.Build).

The builder initially builds a naive SSA form in which all local variables are addresses of stack locations with explicit loads and stores. Registerisation of eligible locals and φ-node insertion using dominance and dataflow are then performed as a second pass called "lifting" to improve the accuracy and performance of subsequent analyses; this pass can be skipped by setting the NaiveForm builder flag.

The primary interfaces of this package are:

A computation that yields a result implements both the [Value](#Value) and [Instruction](#Instruction) interfaces. The following table shows for each concrete type which of these interfaces it implements.

```
Value?          Instruction?      Member?
*Alloc                ✔               ✔
*BinOp                ✔               ✔
*Builtin              ✔
*Call                 ✔               ✔
*ChangeInterface      ✔               ✔
*ChangeType           ✔               ✔
*Const                ✔
*Convert              ✔               ✔
*DebugRef                             ✔
*Defer                                ✔
*Extract              ✔               ✔
*Field                ✔               ✔
*FieldAddr            ✔               ✔
*FreeVar              ✔
*Function             ✔                               ✔ (func)
*Global               ✔                               ✔ (var)
*Go                                   ✔
*If                                   ✔
*Index                ✔               ✔
*IndexAddr            ✔               ✔
*Jump                                 ✔
*Lookup               ✔               ✔
*MakeChan             ✔               ✔
*MakeClosure          ✔               ✔
*MakeInterface        ✔               ✔
*MakeMap              ✔               ✔
*MakeSlice            ✔               ✔
*MapUpdate                            ✔
*MultiConvert         ✔               ✔
*NamedConst                                           ✔ (const)
*Next                 ✔               ✔
*Panic                                ✔
*Parameter            ✔
*Phi                  ✔               ✔
*Range                ✔               ✔
*Return                               ✔
*RunDefers                            ✔
*Select               ✔               ✔
*Send                                 ✔
*Slice                ✔               ✔
*SliceToArrayPointer  ✔               ✔
*Store                                ✔
*Type                                                 ✔ (type)
*TypeAssert           ✔               ✔
*UnOp                 ✔               ✔
```

Other key types in this package include: [Program](#Program), [Package](#Package), [Function](#Function) and [BasicBlock](#BasicBlock).

The program representation constructed by this package is fully resolved internally, i.e. it does not rely on the names of Values, Packages, Functions, Types or BasicBlocks for the correct interpretation of the program. Only the identities of objects and the topology of the SSA and type graphs are semantically significant. (There is one exception: [types.Id](https://pkg.go.dev/go/types#Id) values, which identify field and method names, contain strings.) Avoidance of name-based operations simplifies the implementation of subsequent passes and can make them very efficient. Many objects are nonetheless named to aid in debugging, but it is not essential that the names be either accurate or unambiguous. The public API exposes a number of name-based maps for client convenience.

The [golang.org/x/tools/go/ssa/ssautil](https://pkg.go.dev/golang.org/x/tools@v0.48.0/go/ssa/ssautil) package provides various helper functions, for example to simplify loading a Go program into SSA form.

TODO(adonovan): write a how-to document for all the various cases of trying to determine corresponding elements across the four domains of source locations, ast.Nodes, types.Objects, ssa.Values/Instructions.

Example (BuildPackage) [¶](#example-package-BuildPackage "Go to Example (BuildPackage)")

This program demonstrates how to run the SSA builder on a single package of one or more already-parsed files. Its dependencies are loaded from compiler export data. This is what you'd typically use for a compiler; it does not depend on the obsolete [golang.org/x/tools/go/loader](https://pkg.go.dev/golang.org/x/tools@v0.48.0/go/loader).

It shows the printed representation of packages, functions, and instructions. Within the function listing, the name of each BasicBlock such as ".0.entry" is printed left-aligned, followed by the block's Instructions.

For each instruction that defines an SSA virtual register (i.e. implements Value), the type of that value is shown in the right column.

Build and run the ssadump.go program if you want a standalone tool with similar functionality. It is located at [golang.org/x/tools/cmd/ssadump](https://pkg.go.dev/golang.org/x/tools@v0.48.0/cmd/ssadump).

Use ssautil.BuildPackage only if you have parsed--but not type-checked--syntax trees. Typically, clients already have typed syntax, perhaps obtained from golang.org/x/tools/go/packages. In that case, see the other examples for simpler approaches.

```
package main

import (
    "fmt"
    "go/ast"
    "go/importer"
    "go/parser"
    "go/token"
    "go/types"
    "os"

    "golang.org/x/tools/go/ssa"
    "golang.org/x/tools/go/ssa/ssautil"
)

const hello = \`
package main

import "fmt"

const message = "Hello, World!"

func main() {
    fmt.Println(message)
}
\`

func main() {
    // Parse the source files.
    fset := token.NewFileSet()
    f, err := parser.ParseFile(fset, "hello.go", hello, parser.ParseComments)
    if err != nil {
        fmt.Print(err) // parse error
        return
    }
    files := []*ast.File{f}

    // Create the type-checker's package.
    pkg := types.NewPackage("hello", "")

    // Type-check the package, load dependencies.
    // Create and build the SSA program.
    hello, _, err := ssautil.BuildPackage(
        &types.Config{Importer: importer.Default()}, fset, pkg, files, ssa.SanityCheckFunctions)
    if err != nil {
        fmt.Print(err) // type error in some package
        return
    }

    // Print out the package.
    hello.WriteTo(os.Stdout)

    // Print out the package-level functions.
    hello.Func("init").WriteTo(os.Stdout)
    hello.Func("main").WriteTo(os.Stdout)

}
```
```
Output:

package hello:
  func  init       func()
  var   init$guard bool
  func  main       func()
  const message    message = "Hello, World!":untyped string

# Name: hello.init
# Package: hello
# Synthetic: package initializer
func init():
0:                                                                entry P:0 S:2
    t0 = *init$guard                                                   bool
    if t0 goto 2 else 1
1:                                                    init.start P:1 S:1 idom:0
    *init$guard = true:bool
    t1 = fmt.init()                                                      ()
    jump 2
2:                                                     init.done P:2 S:0 idom:0
    return

# Name: hello.main
# Package: hello
# Location: hello.go:8:6
func main():
0:                                                                entry P:0 S:0
    t0 = new [1]any (varargs)                                       *[1]any
    t1 = &t0[0:int]                                                    *any
    t2 = make any <- string ("Hello, World!":string)                    any
    *t1 = t2
    t3 = slice t0[:]                                                  []any
    t4 = fmt.Println(t3...)                              (n int, err error)
    return
```

Example (LoadPackages) [¶](#example-package-LoadPackages "Go to Example (LoadPackages)")

This example builds SSA code for a set of packages using the [golang.org/x/tools/go/packages](https://pkg.go.dev/golang.org/x/tools@v0.48.0/go/packages) API. This is what you would typically use for a analysis capable of operating on a single package.

```
package main

import (
    "log"

    "golang.org/x/tools/go/packages"
    "golang.org/x/tools/go/ssa"
    "golang.org/x/tools/go/ssa/ssautil"
)

func main() {
    // Load, parse, and type-check the initial packages.
    cfg := &packages.Config{Mode: packages.LoadSyntax}
    initial, err := packages.Load(cfg, "fmt", "net/http")
    if err != nil {
        log.Fatal(err)
    }

    // Stop if any package had errors.
    // This step is optional; without it, the next step
    // will create SSA for only a subset of packages.
    if packages.PrintErrors(initial) > 0 {
        log.Fatalf("packages contain errors")
    }

    // Create SSA packages for all well-typed packages.
    prog, pkgs := ssautil.Packages(initial, ssa.PrintPackages)
    _ = prog

    // Build SSA code for the well-typed initial packages.
    for _, p := range pkgs {
        if p != nil {
            p.Build()
        }
    }
}
```
```
Output:
```

Example (LoadWholeProgram) [¶](#example-package-LoadWholeProgram "Go to Example (LoadWholeProgram)")

This example builds SSA code for a set of packages plus all their dependencies, using the [golang.org/x/tools/go/packages](https://pkg.go.dev/golang.org/x/tools@v0.48.0/go/packages) API. This is what you'd typically use for a whole-program analysis.

```
package main

import (
    "log"

    "golang.org/x/tools/go/packages"
    "golang.org/x/tools/go/ssa"
    "golang.org/x/tools/go/ssa/ssautil"
)

func main() {
    // Load, parse, and type-check the whole program.
    cfg := packages.Config{Mode: packages.LoadAllSyntax}
    initial, err := packages.Load(&cfg, "fmt", "net/http")
    if err != nil {
        log.Fatal(err)
    }

    // Create SSA packages for well-typed packages and their dependencies.
    prog, pkgs := ssautil.AllPackages(initial, ssa.PrintPackages|ssa.InstantiateGenerics)
    _ = pkgs

    // Build SSA code for the whole program.
    prog.Build()
}
```
```
Output:
```

### Index

- [Constants](#pkg-constants)
- [func HasEnclosingFunction(pkg \*Package, path \[\]ast.Node) bool](#HasEnclosingFunction)
- [func WriteFunction(buf \*bytes.Buffer, f \*Function)](#WriteFunction)
- [func WritePackage(buf \*bytes.Buffer, p \*Package)](#WritePackage)
- [type Alloc](#Alloc)
- - [func (v \*Alloc) Name() string](#Alloc.Name)
		- [func (v \*Alloc) Operands(rands \[\]\*Value) \[\]\*Value](#Alloc.Operands)
		- [func (v \*Alloc) Pos() token.Pos](#Alloc.Pos)
		- [func (v \*Alloc) Referrers() \*\[\]Instruction](#Alloc.Referrers)
		- [func (v \*Alloc) String() string](#Alloc.String)
		- [func (v \*Alloc) Type() types.Type](#Alloc.Type)
- [type BasicBlock](#BasicBlock)
- - [func (b \*BasicBlock) Dominates(c \*BasicBlock) bool](#BasicBlock.Dominates)
		- [func (b \*BasicBlock) Dominees() \[\]\*BasicBlock](#BasicBlock.Dominees)
		- [func (b \*BasicBlock) Idom() \*BasicBlock](#BasicBlock.Idom)
		- [func (b \*BasicBlock) Parent() \*Function](#BasicBlock.Parent)
		- [func (b \*BasicBlock) String() string](#BasicBlock.String)
- [type BinOp](#BinOp)
- - [func (v \*BinOp) Name() string](#BinOp.Name)
		- [func (v \*BinOp) Operands(rands \[\]\*Value) \[\]\*Value](#BinOp.Operands)
		- [func (v \*BinOp) Pos() token.Pos](#BinOp.Pos)
		- [func (v \*BinOp) Referrers() \*\[\]Instruction](#BinOp.Referrers)
		- [func (v \*BinOp) String() string](#BinOp.String)
		- [func (v \*BinOp) Type() types.Type](#BinOp.Type)
- [type BuilderMode](#BuilderMode)
- - [func (m BuilderMode) Get() any](#BuilderMode.Get)
		- [func (m \*BuilderMode) Set(s string) error](#BuilderMode.Set)
		- [func (m BuilderMode) String() string](#BuilderMode.String)
- [type Builtin](#Builtin)
- - [func (v \*Builtin) Name() string](#Builtin.Name)
		- [func (v \*Builtin) Object() types.Object](#Builtin.Object)
		- [func (v \*Builtin) Operands(rands \[\]\*Value) \[\]\*Value](#Builtin.Operands)
		- [func (v \*Builtin) Parent() \*Function](#Builtin.Parent)
		- [func (v \*Builtin) Pos() token.Pos](#Builtin.Pos)
		- [func (\*Builtin) Referrers() \*\[\]Instruction](#Builtin.Referrers)
		- [func (v \*Builtin) String() string](#Builtin.String)
		- [func (v \*Builtin) Type() types.Type](#Builtin.Type)
- [type Call](#Call)
- - [func (s \*Call) Common() \*CallCommon](#Call.Common)
		- [func (v \*Call) Name() string](#Call.Name)
		- [func (s \*Call) Operands(rands \[\]\*Value) \[\]\*Value](#Call.Operands)
		- [func (v \*Call) Pos() token.Pos](#Call.Pos)
		- [func (v \*Call) Referrers() \*\[\]Instruction](#Call.Referrers)
		- [func (v \*Call) String() string](#Call.String)
		- [func (v \*Call) Type() types.Type](#Call.Type)
		- [func (s \*Call) Value() \*Call](#Call.Value)
- [type CallCommon](#CallCommon)
- - [func (c \*CallCommon) Description() string](#CallCommon.Description)
		- [func (c \*CallCommon) IsInvoke() bool](#CallCommon.IsInvoke)
		- [func (c \*CallCommon) Operands(rands \[\]\*Value) \[\]\*Value](#CallCommon.Operands)
		- [func (c \*CallCommon) Pos() token.Pos](#CallCommon.Pos)
		- [func (c \*CallCommon) Signature() \*types.Signature](#CallCommon.Signature)
		- [func (c \*CallCommon) StaticCallee() \*Function](#CallCommon.StaticCallee)
		- [func (c \*CallCommon) String() string](#CallCommon.String)
- [type CallInstruction](#CallInstruction)
- [type ChangeInterface](#ChangeInterface)
- - [func (v \*ChangeInterface) Name() string](#ChangeInterface.Name)
		- [func (v \*ChangeInterface) Operands(rands \[\]\*Value) \[\]\*Value](#ChangeInterface.Operands)
		- [func (v \*ChangeInterface) Pos() token.Pos](#ChangeInterface.Pos)
		- [func (v \*ChangeInterface) Referrers() \*\[\]Instruction](#ChangeInterface.Referrers)
		- [func (v \*ChangeInterface) String() string](#ChangeInterface.String)
		- [func (v \*ChangeInterface) Type() types.Type](#ChangeInterface.Type)
- [type ChangeType](#ChangeType)
- - [func (v \*ChangeType) Name() string](#ChangeType.Name)
		- [func (v \*ChangeType) Operands(rands \[\]\*Value) \[\]\*Value](#ChangeType.Operands)
		- [func (v \*ChangeType) Pos() token.Pos](#ChangeType.Pos)
		- [func (v \*ChangeType) Referrers() \*\[\]Instruction](#ChangeType.Referrers)
		- [func (v \*ChangeType) String() string](#ChangeType.String)
		- [func (v \*ChangeType) Type() types.Type](#ChangeType.Type)
- [type Const](#Const)
- - [func NewConst(val constant.Value, typ types.Type) \*Const](#NewConst)
- - [func (c \*Const) Complex128() complex128](#Const.Complex128)
		- [func (c \*Const) Float64() float64](#Const.Float64)
		- [func (c \*Const) Int64() int64](#Const.Int64)
		- [func (c \*Const) IsNil() bool](#Const.IsNil)
		- [func (c \*Const) Name() string](#Const.Name)
		- [func (v \*Const) Operands(rands \[\]\*Value) \[\]\*Value](#Const.Operands)
		- [func (c \*Const) Parent() \*Function](#Const.Parent)
		- [func (c \*Const) Pos() token.Pos](#Const.Pos)
		- [func (c \*Const) Referrers() \*\[\]Instruction](#Const.Referrers)
		- [func (c \*Const) RelString(from \*types.Package) string](#Const.RelString)
		- [func (c \*Const) String() string](#Const.String)
		- [func (c \*Const) Type() types.Type](#Const.Type)
		- [func (c \*Const) Uint64() uint64](#Const.Uint64)
- [type Convert](#Convert)
- - [func (v \*Convert) Name() string](#Convert.Name)
		- [func (v \*Convert) Operands(rands \[\]\*Value) \[\]\*Value](#Convert.Operands)
		- [func (v \*Convert) Pos() token.Pos](#Convert.Pos)
		- [func (v \*Convert) Referrers() \*\[\]Instruction](#Convert.Referrers)
		- [func (v \*Convert) String() string](#Convert.String)
		- [func (v \*Convert) Type() types.Type](#Convert.Type)
- [type DebugRef](#DebugRef)
- - [func (v \*DebugRef) Block() \*BasicBlock](#DebugRef.Block)
		- [func (d \*DebugRef) Object() types.Object](#DebugRef.Object)
		- [func (s \*DebugRef) Operands(rands \[\]\*Value) \[\]\*Value](#DebugRef.Operands)
		- [func (v \*DebugRef) Parent() \*Function](#DebugRef.Parent)
		- [func (s \*DebugRef) Pos() token.Pos](#DebugRef.Pos)
		- [func (v \*DebugRef) Referrers() \*\[\]Instruction](#DebugRef.Referrers)
		- [func (s \*DebugRef) String() string](#DebugRef.String)
- [type Defer](#Defer)
- - [func (v \*Defer) Block() \*BasicBlock](#Defer.Block)
		- [func (s \*Defer) Common() \*CallCommon](#Defer.Common)
		- [func (s \*Defer) Operands(rands \[\]\*Value) \[\]\*Value](#Defer.Operands)
		- [func (v \*Defer) Parent() \*Function](#Defer.Parent)
		- [func (s \*Defer) Pos() token.Pos](#Defer.Pos)
		- [func (v \*Defer) Referrers() \*\[\]Instruction](#Defer.Referrers)
		- [func (s \*Defer) String() string](#Defer.String)
		- [func (s \*Defer) Value() \*Call](#Defer.Value)
- [type Extract](#Extract)
- - [func (v \*Extract) Name() string](#Extract.Name)
		- [func (v \*Extract) Operands(rands \[\]\*Value) \[\]\*Value](#Extract.Operands)
		- [func (v \*Extract) Pos() token.Pos](#Extract.Pos)
		- [func (v \*Extract) Referrers() \*\[\]Instruction](#Extract.Referrers)
		- [func (v \*Extract) String() string](#Extract.String)
		- [func (v \*Extract) Type() types.Type](#Extract.Type)
- [type Field](#Field)
- - [func (v \*Field) Name() string](#Field.Name)
		- [func (v \*Field) Operands(rands \[\]\*Value) \[\]\*Value](#Field.Operands)
		- [func (v \*Field) Pos() token.Pos](#Field.Pos)
		- [func (v \*Field) Referrers() \*\[\]Instruction](#Field.Referrers)
		- [func (v \*Field) String() string](#Field.String)
		- [func (v \*Field) Type() types.Type](#Field.Type)
- [type FieldAddr](#FieldAddr)
- - [func (v \*FieldAddr) Name() string](#FieldAddr.Name)
		- [func (v \*FieldAddr) Operands(rands \[\]\*Value) \[\]\*Value](#FieldAddr.Operands)
		- [func (v \*FieldAddr) Pos() token.Pos](#FieldAddr.Pos)
		- [func (v \*FieldAddr) Referrers() \*\[\]Instruction](#FieldAddr.Referrers)
		- [func (v \*FieldAddr) String() string](#FieldAddr.String)
		- [func (v \*FieldAddr) Type() types.Type](#FieldAddr.Type)
- [type FreeVar](#FreeVar)
- - [func (v \*FreeVar) Name() string](#FreeVar.Name)
		- [func (v \*FreeVar) Operands(rands \[\]\*Value) \[\]\*Value](#FreeVar.Operands)
		- [func (v \*FreeVar) Parent() \*Function](#FreeVar.Parent)
		- [func (v \*FreeVar) Pos() token.Pos](#FreeVar.Pos)
		- [func (v \*FreeVar) Referrers() \*\[\]Instruction](#FreeVar.Referrers)
		- [func (v \*FreeVar) String() string](#FreeVar.String)
		- [func (v \*FreeVar) Type() types.Type](#FreeVar.Type)
- [type Function](#Function)
- - [func EnclosingFunction(pkg \*Package, path \[\]ast.Node) \*Function](#EnclosingFunction)
- - [func (f \*Function) DomPostorder() \[\]\*BasicBlock](#Function.DomPostorder)
		- [func (f \*Function) DomPreorder() \[\]\*BasicBlock](#Function.DomPreorder)
		- [func (v \*Function) Name() string](#Function.Name)
		- [func (v \*Function) Object() types.Object](#Function.Object)
		- [func (v \*Function) Operands(rands \[\]\*Value) \[\]\*Value](#Function.Operands)
		- [func (fn \*Function) Origin() \*Function](#Function.Origin)
		- [func (v \*Function) Package() \*Package](#Function.Package)
		- [func (v \*Function) Parent() \*Function](#Function.Parent)
		- [func (v \*Function) Pos() token.Pos](#Function.Pos)
		- [func (v \*Function) Referrers() \*\[\]Instruction](#Function.Referrers)
		- [func (f \*Function) RelString(from \*types.Package) string](#Function.RelString)
		- [func (v \*Function) String() string](#Function.String)
		- [func (f \*Function) Syntax() ast.Node](#Function.Syntax)
		- [func (v \*Function) Token() token.Token](#Function.Token)
		- [func (v \*Function) Type() types.Type](#Function.Type)
		- [func (fn \*Function) TypeArgs() \[\]types.Type](#Function.TypeArgs)
		- [func (fn \*Function) TypeParams() \*types.TypeParamList](#Function.TypeParams)
		- [func (f \*Function) ValueForExpr(e ast.Expr) (value Value, isAddr bool)](#Function.ValueForExpr)
		- [func (f \*Function) WriteTo(w io.Writer) (int64, error)](#Function.WriteTo)
- [type Global](#Global)
- - [func (v \*Global) Name() string](#Global.Name)
		- [func (v \*Global) Object() types.Object](#Global.Object)
		- [func (v \*Global) Operands(rands \[\]\*Value) \[\]\*Value](#Global.Operands)
		- [func (v \*Global) Package() \*Package](#Global.Package)
		- [func (v \*Global) Parent() \*Function](#Global.Parent)
		- [func (v \*Global) Pos() token.Pos](#Global.Pos)
		- [func (v \*Global) Referrers() \*\[\]Instruction](#Global.Referrers)
		- [func (v \*Global) RelString(from \*types.Package) string](#Global.RelString)
		- [func (v \*Global) String() string](#Global.String)
		- [func (v \*Global) Token() token.Token](#Global.Token)
		- [func (v \*Global) Type() types.Type](#Global.Type)
- [type Go](#Go)
- - [func (v \*Go) Block() \*BasicBlock](#Go.Block)
		- [func (s \*Go) Common() \*CallCommon](#Go.Common)
		- [func (s \*Go) Operands(rands \[\]\*Value) \[\]\*Value](#Go.Operands)
		- [func (v \*Go) Parent() \*Function](#Go.Parent)
		- [func (s \*Go) Pos() token.Pos](#Go.Pos)
		- [func (v \*Go) Referrers() \*\[\]Instruction](#Go.Referrers)
		- [func (s \*Go) String() string](#Go.String)
		- [func (s \*Go) Value() \*Call](#Go.Value)
- [type If](#If)
- - [func (v \*If) Block() \*BasicBlock](#If.Block)
		- [func (s \*If) Operands(rands \[\]\*Value) \[\]\*Value](#If.Operands)
		- [func (v \*If) Parent() \*Function](#If.Parent)
		- [func (s \*If) Pos() token.Pos](#If.Pos)
		- [func (v \*If) Referrers() \*\[\]Instruction](#If.Referrers)
		- [func (s \*If) String() string](#If.String)
- [type Index](#Index)
- - [func (v \*Index) Name() string](#Index.Name)
		- [func (v \*Index) Operands(rands \[\]\*Value) \[\]\*Value](#Index.Operands)
		- [func (v \*Index) Pos() token.Pos](#Index.Pos)
		- [func (v \*Index) Referrers() \*\[\]Instruction](#Index.Referrers)
		- [func (v \*Index) String() string](#Index.String)
		- [func (v \*Index) Type() types.Type](#Index.Type)
- [type IndexAddr](#IndexAddr)
- - [func (v \*IndexAddr) Name() string](#IndexAddr.Name)
		- [func (v \*IndexAddr) Operands(rands \[\]\*Value) \[\]\*Value](#IndexAddr.Operands)
		- [func (v \*IndexAddr) Pos() token.Pos](#IndexAddr.Pos)
		- [func (v \*IndexAddr) Referrers() \*\[\]Instruction](#IndexAddr.Referrers)
		- [func (v \*IndexAddr) String() string](#IndexAddr.String)
		- [func (v \*IndexAddr) Type() types.Type](#IndexAddr.Type)
- [type Instruction](#Instruction)
- [type Jump](#Jump)
- - [func (v \*Jump) Block() \*BasicBlock](#Jump.Block)
		- [func (\*Jump) Operands(rands \[\]\*Value) \[\]\*Value](#Jump.Operands)
		- [func (v \*Jump) Parent() \*Function](#Jump.Parent)
		- [func (s \*Jump) Pos() token.Pos](#Jump.Pos)
		- [func (v \*Jump) Referrers() \*\[\]Instruction](#Jump.Referrers)
		- [func (s \*Jump) String() string](#Jump.String)
- [type Lookup](#Lookup)
- - [func (v \*Lookup) Name() string](#Lookup.Name)
		- [func (v \*Lookup) Operands(rands \[\]\*Value) \[\]\*Value](#Lookup.Operands)
		- [func (v \*Lookup) Pos() token.Pos](#Lookup.Pos)
		- [func (v \*Lookup) Referrers() \*\[\]Instruction](#Lookup.Referrers)
		- [func (v \*Lookup) String() string](#Lookup.String)
		- [func (v \*Lookup) Type() types.Type](#Lookup.Type)
- [type MakeChan](#MakeChan)
- - [func (v \*MakeChan) Name() string](#MakeChan.Name)
		- [func (v \*MakeChan) Operands(rands \[\]\*Value) \[\]\*Value](#MakeChan.Operands)
		- [func (v \*MakeChan) Pos() token.Pos](#MakeChan.Pos)
		- [func (v \*MakeChan) Referrers() \*\[\]Instruction](#MakeChan.Referrers)
		- [func (v \*MakeChan) String() string](#MakeChan.String)
		- [func (v \*MakeChan) Type() types.Type](#MakeChan.Type)
- [type MakeClosure](#MakeClosure)
- - [func (v \*MakeClosure) Name() string](#MakeClosure.Name)
		- [func (v \*MakeClosure) Operands(rands \[\]\*Value) \[\]\*Value](#MakeClosure.Operands)
		- [func (v \*MakeClosure) Pos() token.Pos](#MakeClosure.Pos)
		- [func (v \*MakeClosure) Referrers() \*\[\]Instruction](#MakeClosure.Referrers)
		- [func (v \*MakeClosure) String() string](#MakeClosure.String)
		- [func (v \*MakeClosure) Type() types.Type](#MakeClosure.Type)
- [type MakeInterface](#MakeInterface)
- - [func (v \*MakeInterface) Name() string](#MakeInterface.Name)
		- [func (v \*MakeInterface) Operands(rands \[\]\*Value) \[\]\*Value](#MakeInterface.Operands)
		- [func (v \*MakeInterface) Pos() token.Pos](#MakeInterface.Pos)
		- [func (v \*MakeInterface) Referrers() \*\[\]Instruction](#MakeInterface.Referrers)
		- [func (v \*MakeInterface) String() string](#MakeInterface.String)
		- [func (v \*MakeInterface) Type() types.Type](#MakeInterface.Type)
- [type MakeMap](#MakeMap)
- - [func (v \*MakeMap) Name() string](#MakeMap.Name)
		- [func (v \*MakeMap) Operands(rands \[\]\*Value) \[\]\*Value](#MakeMap.Operands)
		- [func (v \*MakeMap) Pos() token.Pos](#MakeMap.Pos)
		- [func (v \*MakeMap) Referrers() \*\[\]Instruction](#MakeMap.Referrers)
		- [func (v \*MakeMap) String() string](#MakeMap.String)
		- [func (v \*MakeMap) Type() types.Type](#MakeMap.Type)
- [type MakeSlice](#MakeSlice)
- - [func (v \*MakeSlice) Name() string](#MakeSlice.Name)
		- [func (v \*MakeSlice) Operands(rands \[\]\*Value) \[\]\*Value](#MakeSlice.Operands)
		- [func (v \*MakeSlice) Pos() token.Pos](#MakeSlice.Pos)
		- [func (v \*MakeSlice) Referrers() \*\[\]Instruction](#MakeSlice.Referrers)
		- [func (v \*MakeSlice) String() string](#MakeSlice.String)
		- [func (v \*MakeSlice) Type() types.Type](#MakeSlice.Type)
- [type MapUpdate](#MapUpdate)
- - [func (v \*MapUpdate) Block() \*BasicBlock](#MapUpdate.Block)
		- [func (v \*MapUpdate) Operands(rands \[\]\*Value) \[\]\*Value](#MapUpdate.Operands)
		- [func (v \*MapUpdate) Parent() \*Function](#MapUpdate.Parent)
		- [func (s \*MapUpdate) Pos() token.Pos](#MapUpdate.Pos)
		- [func (v \*MapUpdate) Referrers() \*\[\]Instruction](#MapUpdate.Referrers)
		- [func (s \*MapUpdate) String() string](#MapUpdate.String)
- [type Member](#Member)
- [type MultiConvert](#MultiConvert)
- - [func (v \*MultiConvert) Name() string](#MultiConvert.Name)
		- [func (v \*MultiConvert) Operands(rands \[\]\*Value) \[\]\*Value](#MultiConvert.Operands)
		- [func (v \*MultiConvert) Pos() token.Pos](#MultiConvert.Pos)
		- [func (v \*MultiConvert) Referrers() \*\[\]Instruction](#MultiConvert.Referrers)
		- [func (v \*MultiConvert) String() string](#MultiConvert.String)
		- [func (v \*MultiConvert) Type() types.Type](#MultiConvert.Type)
- [type NamedConst](#NamedConst)
- - [func (c \*NamedConst) Name() string](#NamedConst.Name)
		- [func (c \*NamedConst) Object() types.Object](#NamedConst.Object)
		- [func (c \*NamedConst) Package() \*Package](#NamedConst.Package)
		- [func (c \*NamedConst) Pos() token.Pos](#NamedConst.Pos)
		- [func (c \*NamedConst) RelString(from \*types.Package) string](#NamedConst.RelString)
		- [func (c \*NamedConst) String() string](#NamedConst.String)
		- [func (c \*NamedConst) Token() token.Token](#NamedConst.Token)
		- [func (c \*NamedConst) Type() types.Type](#NamedConst.Type)
- [type Next](#Next)
- - [func (v \*Next) Name() string](#Next.Name)
		- [func (v \*Next) Operands(rands \[\]\*Value) \[\]\*Value](#Next.Operands)
		- [func (v \*Next) Pos() token.Pos](#Next.Pos)
		- [func (v \*Next) Referrers() \*\[\]Instruction](#Next.Referrers)
		- [func (v \*Next) String() string](#Next.String)
		- [func (v \*Next) Type() types.Type](#Next.Type)
- [type Node](#Node)
- [type Package](#Package)
- - [func (p \*Package) Build()](#Package.Build)
		- [func (p \*Package) Const(name string) (c \*NamedConst)](#Package.Const)
		- [func (p \*Package) Func(name string) (f \*Function)](#Package.Func)
		- [func (pkg \*Package) SetDebugMode(debug bool)](#Package.SetDebugMode)
		- [func (p \*Package) String() string](#Package.String)
		- [func (p \*Package) Type(name string) (t \*Type)](#Package.Type)
		- [func (p \*Package) Var(name string) (g \*Global)](#Package.Var)
		- [func (p \*Package) WriteTo(w io.Writer) (int64, error)](#Package.WriteTo)
- [type Panic](#Panic)
- - [func (v \*Panic) Block() \*BasicBlock](#Panic.Block)
		- [func (s \*Panic) Operands(rands \[\]\*Value) \[\]\*Value](#Panic.Operands)
		- [func (v \*Panic) Parent() \*Function](#Panic.Parent)
		- [func (s \*Panic) Pos() token.Pos](#Panic.Pos)
		- [func (v \*Panic) Referrers() \*\[\]Instruction](#Panic.Referrers)
		- [func (s \*Panic) String() string](#Panic.String)
- [type Parameter](#Parameter)
- - [func (v \*Parameter) Name() string](#Parameter.Name)
		- [func (v \*Parameter) Object() types.Object](#Parameter.Object)
		- [func (v \*Parameter) Operands(rands \[\]\*Value) \[\]\*Value](#Parameter.Operands)
		- [func (v \*Parameter) Parent() \*Function](#Parameter.Parent)
		- [func (v \*Parameter) Pos() token.Pos](#Parameter.Pos)
		- [func (v \*Parameter) Referrers() \*\[\]Instruction](#Parameter.Referrers)
		- [func (v \*Parameter) String() string](#Parameter.String)
		- [func (v \*Parameter) Type() types.Type](#Parameter.Type)
- [type Phi](#Phi)
- - [func (v \*Phi) Name() string](#Phi.Name)
		- [func (v \*Phi) Operands(rands \[\]\*Value) \[\]\*Value](#Phi.Operands)
		- [func (v \*Phi) Pos() token.Pos](#Phi.Pos)
		- [func (v \*Phi) Referrers() \*\[\]Instruction](#Phi.Referrers)
		- [func (v \*Phi) String() string](#Phi.String)
		- [func (v \*Phi) Type() types.Type](#Phi.Type)
- [type Program](#Program)
- - [func NewProgram(fset \*token.FileSet, mode BuilderMode) \*Program](#NewProgram)
- - [func (prog \*Program) AllPackages() \[\]\*Package](#Program.AllPackages)
		- [func (prog \*Program) Build()](#Program.Build)
		- [func (prog \*Program) ConstValue(obj \*types.Const) \*Const](#Program.ConstValue)
		- [func (prog \*Program) CreatePackage(pkg \*types.Package, files \[\]\*ast.File, info \*types.Info, importable bool) \*Package](#Program.CreatePackage)
		- [func (prog \*Program) FuncValue(obj \*types.Func) \*Function](#Program.FuncValue)
		- [func (prog \*Program) ImportedPackage(path string) \*Package](#Program.ImportedPackage)
		- [func (prog \*Program) LookupMethod(T types.Type, pkg \*types.Package, name string) \*Function](#Program.LookupMethod)
		- [func (prog \*Program) MethodValue(sel \*types.Selection) \*Function](#Program.MethodValue)
		- [func (prog \*Program) NewFunction(name string, sig \*types.Signature, provenance string) \*Function](#Program.NewFunction)
		- [func (prog \*Program) Package(pkg \*types.Package) \*Package](#Program.Package)
		- [func (prog \*Program) RuntimeTypes() \[\]types.Type](#Program.RuntimeTypes)
		- [func (prog \*Program) SetNoReturn(noReturn func(\*types.Func) bool)](#Program.SetNoReturn)
		- [func (prog \*Program) VarValue(obj \*types.Var, pkg \*Package, ref \[\]ast.Node) (value Value, isAddr bool)](#Program.VarValue)
- [type Range](#Range)
- - [func (v \*Range) Name() string](#Range.Name)
		- [func (v \*Range) Operands(rands \[\]\*Value) \[\]\*Value](#Range.Operands)
		- [func (v \*Range) Pos() token.Pos](#Range.Pos)
		- [func (v \*Range) Referrers() \*\[\]Instruction](#Range.Referrers)
		- [func (v \*Range) String() string](#Range.String)
		- [func (v \*Range) Type() types.Type](#Range.Type)
- [type Return](#Return)
- - [func (v \*Return) Block() \*BasicBlock](#Return.Block)
		- [func (s \*Return) Operands(rands \[\]\*Value) \[\]\*Value](#Return.Operands)
		- [func (v \*Return) Parent() \*Function](#Return.Parent)
		- [func (s \*Return) Pos() token.Pos](#Return.Pos)
		- [func (v \*Return) Referrers() \*\[\]Instruction](#Return.Referrers)
		- [func (s \*Return) String() string](#Return.String)
- [type RunDefers](#RunDefers)
- - [func (v \*RunDefers) Block() \*BasicBlock](#RunDefers.Block)
		- [func (\*RunDefers) Operands(rands \[\]\*Value) \[\]\*Value](#RunDefers.Operands)
		- [func (v \*RunDefers) Parent() \*Function](#RunDefers.Parent)
		- [func (s \*RunDefers) Pos() token.Pos](#RunDefers.Pos)
		- [func (v \*RunDefers) Referrers() \*\[\]Instruction](#RunDefers.Referrers)
		- [func (\*RunDefers) String() string](#RunDefers.String)
- [type Select](#Select)
- - [func (v \*Select) Name() string](#Select.Name)
		- [func (v \*Select) Operands(rands \[\]\*Value) \[\]\*Value](#Select.Operands)
		- [func (v \*Select) Pos() token.Pos](#Select.Pos)
		- [func (v \*Select) Referrers() \*\[\]Instruction](#Select.Referrers)
		- [func (s \*Select) String() string](#Select.String)
		- [func (v \*Select) Type() types.Type](#Select.Type)
- [type SelectState](#SelectState)
- [type Send](#Send)
- - [func (v \*Send) Block() \*BasicBlock](#Send.Block)
		- [func (s \*Send) Operands(rands \[\]\*Value) \[\]\*Value](#Send.Operands)
		- [func (v \*Send) Parent() \*Function](#Send.Parent)
		- [func (s \*Send) Pos() token.Pos](#Send.Pos)
		- [func (v \*Send) Referrers() \*\[\]Instruction](#Send.Referrers)
		- [func (s \*Send) String() string](#Send.String)
- [type Slice](#Slice)
- - [func (v \*Slice) Name() string](#Slice.Name)
		- [func (v \*Slice) Operands(rands \[\]\*Value) \[\]\*Value](#Slice.Operands)
		- [func (v \*Slice) Pos() token.Pos](#Slice.Pos)
		- [func (v \*Slice) Referrers() \*\[\]Instruction](#Slice.Referrers)
		- [func (v \*Slice) String() string](#Slice.String)
		- [func (v \*Slice) Type() types.Type](#Slice.Type)
- [type SliceToArrayPointer](#SliceToArrayPointer)
- - [func (v \*SliceToArrayPointer) Name() string](#SliceToArrayPointer.Name)
		- [func (v \*SliceToArrayPointer) Operands(rands \[\]\*Value) \[\]\*Value](#SliceToArrayPointer.Operands)
		- [func (v \*SliceToArrayPointer) Pos() token.Pos](#SliceToArrayPointer.Pos)
		- [func (v \*SliceToArrayPointer) Referrers() \*\[\]Instruction](#SliceToArrayPointer.Referrers)
		- [func (v \*SliceToArrayPointer) String() string](#SliceToArrayPointer.String)
		- [func (v \*SliceToArrayPointer) Type() types.Type](#SliceToArrayPointer.Type)
- [type Store](#Store)
- - [func (v \*Store) Block() \*BasicBlock](#Store.Block)
		- [func (s \*Store) Operands(rands \[\]\*Value) \[\]\*Value](#Store.Operands)
		- [func (v \*Store) Parent() \*Function](#Store.Parent)
		- [func (s \*Store) Pos() token.Pos](#Store.Pos)
		- [func (v \*Store) Referrers() \*\[\]Instruction](#Store.Referrers)
		- [func (s \*Store) String() string](#Store.String)
- [type Type](#Type)
- - [func (t \*Type) Name() string](#Type.Name)
		- [func (t \*Type) Object() types.Object](#Type.Object)
		- [func (t \*Type) Package() \*Package](#Type.Package)
		- [func (t \*Type) Pos() token.Pos](#Type.Pos)
		- [func (t \*Type) RelString(from \*types.Package) string](#Type.RelString)
		- [func (t \*Type) String() string](#Type.String)
		- [func (t \*Type) Token() token.Token](#Type.Token)
		- [func (t \*Type) Type() types.Type](#Type.Type)
- [type TypeAssert](#TypeAssert)
- - [func (v \*TypeAssert) Name() string](#TypeAssert.Name)
		- [func (v \*TypeAssert) Operands(rands \[\]\*Value) \[\]\*Value](#TypeAssert.Operands)
		- [func (v \*TypeAssert) Pos() token.Pos](#TypeAssert.Pos)
		- [func (v \*TypeAssert) Referrers() \*\[\]Instruction](#TypeAssert.Referrers)
		- [func (v \*TypeAssert) String() string](#TypeAssert.String)
		- [func (v \*TypeAssert) Type() types.Type](#TypeAssert.Type)
- [type UnOp](#UnOp)
- - [func (v \*UnOp) Name() string](#UnOp.Name)
		- [func (v \*UnOp) Operands(rands \[\]\*Value) \[\]\*Value](#UnOp.Operands)
		- [func (v \*UnOp) Pos() token.Pos](#UnOp.Pos)
		- [func (v \*UnOp) Referrers() \*\[\]Instruction](#UnOp.Referrers)
		- [func (v \*UnOp) String() string](#UnOp.String)
		- [func (v \*UnOp) Type() types.Type](#UnOp.Type)
- [type Value](#Value)

### Examples

### Constants

```
const BuilderModeDoc = \`\` /* 567-byte string literal not displayed */
```

### Variables

This section is empty.

### Functions

#### func HasEnclosingFunction ¶

```
func HasEnclosingFunction(pkg *Package, path []ast.Node) bool
```

HasEnclosingFunction returns true if the AST node denoted by path is contained within the declaration of some function or package-level variable.

Unlike EnclosingFunction, the behaviour of this function does not depend on whether SSA code for pkg has been built, so it can be used to quickly reject check inputs that will cause EnclosingFunction to fail, prior to SSA building.

#### func WriteFunction ¶

```
func WriteFunction(buf *bytes.Buffer, f *Function)
```

WriteFunction writes to buf a human-readable "disassembly" of f.

#### func WritePackage ¶

```
func WritePackage(buf *bytes.Buffer, p *Package)
```

WritePackage writes to buf a human-readable summary of p.

### Types

#### type Alloc ¶

```
type Alloc struct {
    Comment string
    Heap    bool
    // contains filtered or unexported fields
}
```

The Alloc instruction reserves space for a variable of the given type, zero-initializes it, and yields its address.

Alloc values are always addresses, and have pointer types, so the type of the allocated variable is actually Type().Underlying().(\*types.Pointer).Elem().

If Heap is false, Alloc zero-initializes the same local variable in the call frame and returns its address; in this case the Alloc must be present in Function.Locals. We call this a "local" alloc.

If Heap is true, Alloc allocates a new zero-initialized variable each time the instruction is executed. We call this a "new" alloc.

When Alloc is applied to a channel, map or slice type, it returns the address of an uninitialized (nil) reference of that kind; store the result of MakeSlice, MakeMap or MakeChan in that location to instantiate these types.

Pos() returns the ast.CompositeLit.Lbrace for a composite literal, or the ast.CallExpr.Rparen for a call to new() or for a call that allocates a varargs slice.

Example printed form:

```
t0 = local int
t1 = new int
```

#### func (\*Alloc) Name ¶

```
func (v *Alloc) Name() string
```

#### func (\*Alloc) Operands ¶

```
func (v *Alloc) Operands(rands []*Value) []*Value
```

#### func (\*Alloc) Pos ¶

```
func (v *Alloc) Pos() token.Pos
```

#### func (\*Alloc) Referrers ¶

```
func (v *Alloc) Referrers() *[]Instruction
```

#### func (\*Alloc) String ¶

```
func (v *Alloc) String() string
```

#### func (\*Alloc) Type ¶

```
func (v *Alloc) Type() types.Type
```

#### type BasicBlock ¶

```
type BasicBlock struct {
    Index   int    // index of this block within Parent().Blocks
    Comment string // optional label; no semantic significance

    Instrs       []Instruction // instructions in order
    Preds, Succs []*BasicBlock // predecessors and successors
    // contains filtered or unexported fields
}
```

BasicBlock represents an SSA basic block.

The final element of Instrs is always an explicit transfer of control (If, Jump, Return, or Panic).

A block may contain no Instructions only if it is unreachable, i.e., Preds is nil. Empty blocks are typically pruned.

BasicBlocks and their Preds/Succs relation form a (possibly cyclic) graph independent of the SSA Value graph: the control-flow graph or CFG. It is illegal for multiple edges to exist between the same pair of blocks.

Each BasicBlock is also a node in the dominator tree of the CFG. The tree may be navigated using Idom()/Dominees() and queried using Dominates().

The order of Preds and Succs is significant (to Phi and If instructions, respectively).

#### func (\*BasicBlock) Dominates ¶

```
func (b *BasicBlock) Dominates(c *BasicBlock) bool
```

Dominates reports whether b dominates c.

#### func (\*BasicBlock) Dominees ¶

```
func (b *BasicBlock) Dominees() []*BasicBlock
```

Dominees returns the list of blocks that b immediately dominates: its children in the dominator tree.

#### func (\*BasicBlock) Idom ¶

```
func (b *BasicBlock) Idom() *BasicBlock
```

Idom returns the block that immediately dominates b: its parent in the dominator tree, if any. Neither the entry node (b.Index==0) nor recover node (b==b.Parent().Recover()) have a parent.

#### func (\*BasicBlock) Parent ¶

```
func (b *BasicBlock) Parent() *Function
```

Parent returns the function that contains block b.

#### func (\*BasicBlock) String ¶

```
func (b *BasicBlock) String() string
```

String returns a human-readable label of this block. It is not guaranteed unique within the function.

#### type BinOp ¶

```
type BinOp struct {

    // One of:
    // ADD SUB MUL QUO REM          + - * / %
    // AND OR XOR SHL SHR AND_NOT   & | ^ << >> &^
    // EQL NEQ LSS LEQ GTR GEQ      == != < <= < >=
    Op   token.Token
    X, Y Value
    // contains filtered or unexported fields
}
```

The BinOp instruction yields the result of binary operation X Op Y.

Pos() returns the ast.BinaryExpr.OpPos, if explicit in the source.

Example printed form:

```
t1 = t0 + 1:int
```

#### func (\*BinOp) Name ¶

```
func (v *BinOp) Name() string
```

#### func (\*BinOp) Operands ¶

```
func (v *BinOp) Operands(rands []*Value) []*Value
```

#### func (\*BinOp) Pos ¶

```
func (v *BinOp) Pos() token.Pos
```

#### func (\*BinOp) Referrers ¶

```
func (v *BinOp) Referrers() *[]Instruction
```

#### func (\*BinOp) String ¶

```
func (v *BinOp) String() string
```

#### func (\*BinOp) Type ¶

```
func (v *BinOp) Type() types.Type
```

#### type BuilderMode ¶

```
type BuilderMode uint
```

BuilderMode is a bitmask of options for diagnostics and checking.

\*BuilderMode satisfies the flag.Value interface. Example:

```
var mode = ssa.BuilderMode(0)
func init() { flag.Var(&mode, "build", ssa.BuilderModeDoc) }
```

```
const (
    PrintPackages        BuilderMode = 1 << iota // Print package inventory to stdout
    PrintFunctions                               // Print function SSA code to stdout
    LogSource                                    // Log source locations as SSA builder progresses
    SanityCheckFunctions                         // Perform sanity checking of function bodies
    NaiveForm                                    // Build naïve SSA form: don't replace local loads/stores with registers
    BuildSerially                                // Build packages serially, not in parallel.
    GlobalDebug                                  // Enable debug info for all packages
    BareInits                                    // Build init functions without guards or calls to dependent inits
    InstantiateGenerics                          // Instantiate generics functions (monomorphize) while building
)
```

#### func (BuilderMode) Get ¶

```
func (m BuilderMode) Get() any
```

Get returns m.

#### func (\*BuilderMode) Set ¶

```
func (m *BuilderMode) Set(s string) error
```

Set parses the flag characters in s and updates \*m.

#### func (BuilderMode) String ¶

```
func (m BuilderMode) String() string
```

#### type Builtin ¶

```
type Builtin struct {
    // contains filtered or unexported fields
}
```

A Builtin represents a specific use of a built-in function, e.g. len.

Builtins are immutable values. Builtins do not have addresses. Builtins can only appear in CallCommon.Value.

Name() indicates the function: one of the built-in functions from the Go spec (excluding "make" and "new") or one of these ssa-defined intrinsics:

```
// wrapnilchk returns ptr if non-nil, panics otherwise.
// (For use in indirection wrappers.)
func ssa:wrapnilchk(ptr *T, recvType, methodName string) *T
```

Object() returns a \*types.Builtin for built-ins defined by the spec, nil for others.

Type() returns a \*types.Signature representing the effective signature of the built-in for this call.

#### func (\*Builtin) Name ¶

```
func (v *Builtin) Name() string
```

#### func (\*Builtin) Object ¶

```
func (v *Builtin) Object() types.Object
```

#### func (\*Builtin) Operands ¶

```
func (v *Builtin) Operands(rands []*Value) []*Value
```

Non-Instruction Values:

#### func (\*Builtin) Parent ¶

```
func (v *Builtin) Parent() *Function
```

#### func (\*Builtin) Pos ¶

```
func (v *Builtin) Pos() token.Pos
```

#### func (\*Builtin) Referrers ¶

```
func (*Builtin) Referrers() *[]Instruction
```

#### func (\*Builtin) String ¶

```
func (v *Builtin) String() string
```

#### func (\*Builtin) Type ¶

```
func (v *Builtin) Type() types.Type
```

#### type Call ¶

```
type Call struct {
    Call CallCommon
    // contains filtered or unexported fields
}
```

The Call instruction represents a function or method call.

The Call instruction yields the function result if there is exactly one. Otherwise it returns a tuple, the components of which are accessed via Extract.

See CallCommon for generic function call documentation.

Pos() returns the ast.CallExpr.Lparen, if explicit in the source.

Example printed form:

```
t2 = println(t0, t1)
t4 = t3()
t7 = invoke t5.Println(...t6)
```

#### func (\*Call) Common ¶

```
func (s *Call) Common() *CallCommon
```

#### func (\*Call) Name ¶

```
func (v *Call) Name() string
```

#### func (\*Call) Operands ¶

```
func (s *Call) Operands(rands []*Value) []*Value
```

#### func (\*Call) Pos ¶

```
func (v *Call) Pos() token.Pos
```

#### func (\*Call) Referrers ¶

```
func (v *Call) Referrers() *[]Instruction
```

#### func (\*Call) String ¶

```
func (v *Call) String() string
```

#### func (\*Call) Type ¶

```
func (v *Call) Type() types.Type
```

#### func (\*Call) Value ¶

```
func (s *Call) Value() *Call
```

#### type CallCommon ¶

```
type CallCommon struct {
    Value  Value       // receiver (invoke mode) or func value (call mode)
    Method *types.Func // interface method (invoke mode)
    Args   []Value     // actual parameters (in static method call, includes receiver)
    // contains filtered or unexported fields
}
```

CallCommon is contained by Go, Defer and Call to hold the common parts of a function or method call.

Each CallCommon exists in one of two modes, function call and interface method invocation, or "call" and "invoke" for short.

1\. "call" mode: when Method is nil (!IsInvoke), a CallCommon represents an ordinary function call of the value in Value, which may be a \*Builtin, a \*Function or any other value of kind 'func'.

Value may be one of:

```
(a) a *Function, indicating a statically dispatched call
    to a package-level function, an anonymous function, or
    a method of a named type.
(b) a *MakeClosure, indicating an immediately applied
    function literal with free variables.
(c) a *Builtin, indicating a statically dispatched call
    to a built-in function.
(d) any other value, indicating a dynamically dispatched
    function call.
```

StaticCallee returns the identity of the callee in cases (a) and (b), nil otherwise.

Args contains the arguments to the call. If Value is a method, Args\[0\] contains the receiver parameter.

Example printed form:

```
t2 = println(t0, t1)
go t3()
defer t5(...t6)
```

2\. "invoke" mode: when Method is non-nil (IsInvoke), a CallCommon represents a dynamically dispatched call to an interface method. In this mode, Value is the interface value and Method is the interface's abstract method. The interface value may be a type parameter. Note: an interface method may be shared by multiple interfaces due to embedding; Value.Type() provides the specific interface used for this call.

Value is implicitly supplied to the concrete method implementation as the receiver parameter; in other words, Args\[0\] holds not the receiver but the first true argument.

Example printed form:

```
t1 = invoke t0.String()
go invoke t3.Run(t2)
defer invoke t4.Handle(...t5)
```

For all calls to variadic functions (Signature().Variadic()), the last element of Args is a slice.

#### func (\*CallCommon) Description ¶

```
func (c *CallCommon) Description() string
```

Description returns a description of the mode of this call suitable for a user interface, e.g., "static method call".

#### func (\*CallCommon) IsInvoke ¶

```
func (c *CallCommon) IsInvoke() bool
```

IsInvoke returns true if this call has "invoke" (not "call") mode.

#### func (\*CallCommon) Operands ¶

```
func (c *CallCommon) Operands(rands []*Value) []*Value
```

#### func (\*CallCommon) Pos ¶

```
func (c *CallCommon) Pos() token.Pos
```

#### func (\*CallCommon) Signature ¶

```
func (c *CallCommon) Signature() *types.Signature
```

Signature returns the signature of the called function.

For an "invoke"-mode call, the signature of the interface method is returned.

In either "call" or "invoke" mode, if the callee is a method, its receiver is represented by sig.Recv, not sig.Params().At(0).

#### func (\*CallCommon) StaticCallee ¶

```
func (c *CallCommon) StaticCallee() *Function
```

StaticCallee returns the callee if this is a trivially static "call"-mode call to a function.

#### func (\*CallCommon) String ¶

```
func (c *CallCommon) String() string
```

#### type CallInstruction ¶

```
type CallInstruction interface {
    Instruction
    Common() *CallCommon // returns the common parts of the call
    Value() *Call        // returns the result value of the call (*Call) or nil (*Go, *Defer)
}
```

The CallInstruction interface, implemented by \*Go, \*Defer and \*Call, exposes the common parts of function-calling instructions, yet provides a way back to the Value defined by \*Call alone.

#### type ChangeInterface ¶

```
type ChangeInterface struct {
    X Value
    // contains filtered or unexported fields
}
```

ChangeInterface constructs a value of one interface type from a value of another interface type known to be assignable to it. This operation cannot fail.

Pos() returns the ast.CallExpr.Lparen if the instruction arose from an explicit T(e) conversion; the ast.TypeAssertExpr.Lparen if the instruction arose from an explicit e.(T) operation; or token.NoPos otherwise.

Example printed form:

```
t1 = change interface interface{} <- I (t0)
```

#### func (\*ChangeInterface) Name ¶

```
func (v *ChangeInterface) Name() string
```

#### func (\*ChangeInterface) Operands ¶

```
func (v *ChangeInterface) Operands(rands []*Value) []*Value
```

#### func (\*ChangeInterface) Pos ¶

```
func (v *ChangeInterface) Pos() token.Pos
```

#### func (\*ChangeInterface) Referrers ¶

```
func (v *ChangeInterface) Referrers() *[]Instruction
```

#### func (\*ChangeInterface) String ¶

```
func (v *ChangeInterface) String() string
```

#### func (\*ChangeInterface) Type ¶

```
func (v *ChangeInterface) Type() types.Type
```

#### type ChangeType ¶

```
type ChangeType struct {
    X Value
    // contains filtered or unexported fields
}
```

The ChangeType instruction applies to X a value-preserving type change to Type().

Type changes are permitted:

- between a named type and its underlying type.
- between two named types of the same underlying type.
- between (possibly named) pointers to identical base types.
- from a bidirectional channel to a read- or write-channel, optionally adding/removing a name.
- between a type (t) and an instance of the type (tσ), i.e. Type() == σ(X.Type()) (or X.Type()== σ(Type())) where σ is the type substitution of Parent().TypeParams by Parent().TypeArgs.

This operation cannot fail dynamically.

Type changes may to be to or from a type parameter (or both). All types in the type set of X.Type() have a value-preserving type change to all types in the type set of Type().

Pos() returns the ast.CallExpr.Lparen, if the instruction arose from an explicit conversion in the source.

Example printed form:

```
t1 = changetype *int <- IntPtr (t0)
```

#### func (\*ChangeType) Name ¶

```
func (v *ChangeType) Name() string
```

#### func (\*ChangeType) Operands ¶

```
func (v *ChangeType) Operands(rands []*Value) []*Value
```

#### func (\*ChangeType) Pos ¶

```
func (v *ChangeType) Pos() token.Pos
```

#### func (\*ChangeType) Referrers ¶

```
func (v *ChangeType) Referrers() *[]Instruction
```

#### func (\*ChangeType) String ¶

```
func (v *ChangeType) String() string
```

#### func (\*ChangeType) Type ¶

```
func (v *ChangeType) Type() types.Type
```

#### type Const ¶

```
type Const struct {
    Value constant.Value
    // contains filtered or unexported fields
}
```

A Const represents a value known at build time.

Consts include true constants of boolean, numeric, and string types, as defined by the Go spec; these are represented by a non-nil Value field.

Consts also include the "zero" value of any type, of which the nil values of various pointer-like types are a special case; these are represented by a nil Value field.

Pos() returns token.NoPos.

Example printed forms:

```
42:int
"hello":untyped string
3+4i:MyComplex
nil:*int
nil:[]string
[3]int{}:[3]int
struct{x string}{}:struct{x string}
0:interface{int|int64}
nil:interface{bool|int} // no go/constant representation
```

#### func NewConst ¶

```
func NewConst(val constant.Value, typ types.Type) *Const
```

NewConst returns a new constant of the specified value and type. val must be valid according to the specification of Const.Value.

#### func (\*Const) Complex128 ¶

```
func (c *Const) Complex128() complex128
```

Complex128 returns the complex value of this constant truncated to fit a complex128.

#### func (\*Const) Float64 ¶

```
func (c *Const) Float64() float64
```

Float64 returns the numeric value of this constant truncated to fit a float64.

#### func (\*Const) Int64 ¶

```
func (c *Const) Int64() int64
```

Int64 returns the numeric value of this constant truncated to fit a signed 64-bit integer.

#### func (\*Const) IsNil ¶

```
func (c *Const) IsNil() bool
```

IsNil returns true if this constant is a nil value of a nillable reference type (pointer, slice, channel, map, or function), a basic interface type, or a type parameter all of whose possible instantiations are themselves nillable.

#### func (\*Const) Name ¶

```
func (c *Const) Name() string
```

#### func (\*Const) Operands ¶

```
func (v *Const) Operands(rands []*Value) []*Value
```

#### func (\*Const) Parent ¶

```
func (c *Const) Parent() *Function
```

#### func (\*Const) Pos ¶

```
func (c *Const) Pos() token.Pos
```

#### func (\*Const) Referrers ¶

```
func (c *Const) Referrers() *[]Instruction
```

#### func (\*Const) RelString ¶

```
func (c *Const) RelString(from *types.Package) string
```

#### func (\*Const) String ¶

```
func (c *Const) String() string
```

#### func (\*Const) Type ¶

```
func (c *Const) Type() types.Type
```

#### func (\*Const) Uint64 ¶

```
func (c *Const) Uint64() uint64
```

Uint64 returns the numeric value of this constant truncated to fit an unsigned 64-bit integer.

#### type Convert ¶

```
type Convert struct {
    X Value
    // contains filtered or unexported fields
}
```

The Convert instruction yields the conversion of value X to type Type(). One or both of those types is basic (but possibly named).

A conversion may change the value and representation of its operand. Conversions are permitted:

- between real numeric types.
- between complex numeric types.
- between string and \[\]byte or \[\]rune.
- between pointers and unsafe.Pointer.
- between unsafe.Pointer and uintptr.
- from (Unicode) integer to (UTF-8) string.

A conversion may imply a type name change also.

Conversions may to be to or from a type parameter. All types in the type set of X.Type() can be converted to all types in the type set of Type().

This operation cannot fail dynamically.

Conversions of untyped string/number/bool constants to a specific representation are eliminated during SSA construction.

Pos() returns the ast.CallExpr.Lparen, if the instruction arose from an explicit conversion in the source.

Example printed form:

```
t1 = convert []byte <- string (t0)
```

#### func (\*Convert) Name ¶

```
func (v *Convert) Name() string
```

#### func (\*Convert) Operands ¶

```
func (v *Convert) Operands(rands []*Value) []*Value
```

#### func (\*Convert) Pos ¶

```
func (v *Convert) Pos() token.Pos
```

#### func (\*Convert) Referrers ¶

```
func (v *Convert) Referrers() *[]Instruction
```

#### func (\*Convert) String ¶

```
func (v *Convert) String() string
```

#### func (\*Convert) Type ¶

```
func (v *Convert) Type() types.Type
```

#### type DebugRef ¶

```
type DebugRef struct {
    Expr ast.Expr // the referring expression (never *ast.ParenExpr)

    IsAddr bool  // Expr is addressable and X is the address it denotes
    X      Value // the value or address of Expr
    // contains filtered or unexported fields
}
```

A DebugRef instruction maps a source-level expression Expr to the SSA value X that represents the value (!IsAddr) or address (IsAddr) of that expression.

DebugRef is a pseudo-instruction: it has no dynamic effect.

Pos() returns Expr.Pos(), the start position of the source-level expression. This is not the same as the "designated" token as documented at Value.Pos(). e.g. CallExpr.Pos() does not return the position of the ("designated") Lparen token.

If Expr is an \*ast.Ident denoting a var or func, Object() returns the object; though this information can be obtained from the type checker, including it here greatly facilitates debugging. For non-Ident expressions, Object() returns nil.

DebugRefs are generated only for functions built with debugging enabled; see Package.SetDebugMode() and the GlobalDebug builder mode flag.

DebugRefs are not emitted for ast.Idents referring to constants or predeclared identifiers, since they are trivial and numerous. Nor are they emitted for ast.ParenExprs.

(By representing these as instructions, rather than out-of-band, consistency is maintained during transformation passes by the ordinary SSA renaming machinery.)

Example printed form:

```
; *ast.CallExpr @ 102:9 is t5
; var x float64 @ 109:72 is x
; address of *ast.CompositeLit @ 216:10 is t0
```

#### func (\*DebugRef) Block ¶

```
func (v *DebugRef) Block() *BasicBlock
```

#### func (\*DebugRef) Object ¶

```
func (d *DebugRef) Object() types.Object
```

#### func (\*DebugRef) Operands ¶

```
func (s *DebugRef) Operands(rands []*Value) []*Value
```

#### func (\*DebugRef) Parent ¶

```
func (v *DebugRef) Parent() *Function
```

#### func (\*DebugRef) Pos ¶

```
func (s *DebugRef) Pos() token.Pos
```

#### func (\*DebugRef) Referrers ¶

```
func (v *DebugRef) Referrers() *[]Instruction
```

#### func (\*DebugRef) String ¶

```
func (s *DebugRef) String() string
```

#### type Defer ¶

```
type Defer struct {
    Call       CallCommon
    DeferStack Value // stack of deferred functions (from ssa:deferstack() intrinsic) onto which this function is pushed
    // contains filtered or unexported fields
}
```

The Defer instruction pushes the specified call onto a stack of functions to be called by a RunDefers instruction or by a panic.

If DeferStack!= nil, it indicates the defer list that the defer is added to. Defer list values come from the Builtin function ssa:deferstack. Calls to ssa:deferstack() produces the defer stack of the current function frame. DeferStack allows for deferring into an alternative function stack than the current function.

See CallCommon for generic function call documentation.

Pos() returns the ast.DeferStmt.Defer.

Example printed form:

```
defer println(t0, t1)
defer t3()
defer invoke t5.Println(...t6)
```

#### func (\*Defer) Block ¶

```
func (v *Defer) Block() *BasicBlock
```

#### func (\*Defer) Common ¶

```
func (s *Defer) Common() *CallCommon
```

#### func (\*Defer) Operands ¶

```
func (s *Defer) Operands(rands []*Value) []*Value
```

#### func (\*Defer) Parent ¶

```
func (v *Defer) Parent() *Function
```

#### func (\*Defer) Pos ¶

```
func (s *Defer) Pos() token.Pos
```

#### func (\*Defer) Referrers ¶

```
func (v *Defer) Referrers() *[]Instruction
```

#### func (\*Defer) String ¶

```
func (s *Defer) String() string
```

#### func (\*Defer) Value ¶

```
func (s *Defer) Value() *Call
```

#### type Extract ¶

```
type Extract struct {
    Tuple Value
    Index int
    // contains filtered or unexported fields
}
```

The Extract instruction yields component Index of Tuple.

This is used to access the results of instructions with multiple return values, such as Call, TypeAssert, Next, UnOp(ARROW) and IndexExpr(Map).

Example printed form:

```
t1 = extract t0 #1
```

#### func (\*Extract) Name ¶

```
func (v *Extract) Name() string
```

#### func (\*Extract) Operands ¶

```
func (v *Extract) Operands(rands []*Value) []*Value
```

#### func (\*Extract) Pos ¶

```
func (v *Extract) Pos() token.Pos
```

#### func (\*Extract) Referrers ¶

```
func (v *Extract) Referrers() *[]Instruction
```

#### func (\*Extract) String ¶

```
func (v *Extract) String() string
```

#### func (\*Extract) Type ¶

```
func (v *Extract) Type() types.Type
```

#### type Field ¶

```
type Field struct {
    X     Value // struct
    Field int   // index into CoreType(X.Type()).(*types.Struct).Fields
    // contains filtered or unexported fields
}
```

Example printed form:

```
t1 = t0.name [#1]
```

#### func (\*Field) Name ¶

```
func (v *Field) Name() string
```

#### func (\*Field) Operands ¶

```
func (v *Field) Operands(rands []*Value) []*Value
```

#### func (\*Field) Pos ¶

```
func (v *Field) Pos() token.Pos
```

#### func (\*Field) Referrers ¶

```
func (v *Field) Referrers() *[]Instruction
```

#### func (\*Field) String ¶

```
func (v *Field) String() string
```

#### func (\*Field) Type ¶

```
func (v *Field) Type() types.Type
```

#### type FieldAddr ¶

```
type FieldAddr struct {
    X     Value // *struct
    Field int   // index into CoreType(CoreType(X.Type()).(*types.Pointer).Elem()).(*types.Struct).Fields
    // contains filtered or unexported fields
}
```

The FieldAddr instruction yields the address of Field of \*struct X.

The field is identified by its index within the field list of the struct type of X.

Dynamically, this instruction panics if X evaluates to a nil pointer.

Type() returns a (possibly named) \*types.Pointer.

Pos() returns the position of the ast.SelectorExpr.Sel for the field, if explicit in the source. For implicit selections, returns the position of the inducing explicit selection. If produced for a struct literal S{f: e}, it returns the position of the colon; for S{e} it returns the start of expression e.

Example printed form:

```
t1 = &t0.name [#1]
```

#### func (\*FieldAddr) Name ¶

```
func (v *FieldAddr) Name() string
```

#### func (\*FieldAddr) Operands ¶

```
func (v *FieldAddr) Operands(rands []*Value) []*Value
```

#### func (\*FieldAddr) Pos ¶

```
func (v *FieldAddr) Pos() token.Pos
```

#### func (\*FieldAddr) Referrers ¶

```
func (v *FieldAddr) Referrers() *[]Instruction
```

#### func (\*FieldAddr) String ¶

```
func (v *FieldAddr) String() string
```

#### func (\*FieldAddr) Type ¶

```
func (v *FieldAddr) Type() types.Type
```

#### type FreeVar ¶

```
type FreeVar struct {
    // contains filtered or unexported fields
}
```

A FreeVar represents a free variable of the function to which it belongs.

FreeVars are used to implement anonymous functions, whose free variables are lexically captured in a closure formed by MakeClosure. The value of such a free var is an Alloc or another FreeVar and is considered a potentially escaping heap address, with pointer type.

FreeVars are also used to implement bound method closures. Such a free var represents the receiver value and may be of any type that has concrete methods.

Pos() returns the position of the value that was captured, which belongs to an enclosing function.

#### func (\*FreeVar) Name ¶

```
func (v *FreeVar) Name() string
```

#### func (\*FreeVar) Operands ¶

```
func (v *FreeVar) Operands(rands []*Value) []*Value
```

#### func (\*FreeVar) Parent ¶

```
func (v *FreeVar) Parent() *Function
```

#### func (\*FreeVar) Pos ¶

```
func (v *FreeVar) Pos() token.Pos
```

#### func (\*FreeVar) Referrers ¶

```
func (v *FreeVar) Referrers() *[]Instruction
```

#### func (\*FreeVar) String ¶

```
func (v *FreeVar) String() string
```

#### func (\*FreeVar) Type ¶

```
func (v *FreeVar) Type() types.Type
```

#### type Function ¶

```
type Function struct {
    Signature *types.Signature

    // source information
    Synthetic string // provenance of synthetic function; "" for true source functions

    Pkg  *Package // enclosing package; nil for shared funcs (wrappers and error.Error)
    Prog *Program // enclosing program

    Params    []*Parameter  // function parameters; for methods, includes receiver
    FreeVars  []*FreeVar    // free variables whose values must be supplied by closure
    Locals    []*Alloc      // frame-allocated variables of this function
    Blocks    []*BasicBlock // basic blocks of the function; nil => external
    Recover   *BasicBlock   // optional; control transfers here after recovered panic
    AnonFuncs []*Function   // anonymous functions (from FuncLit,RangeStmt) directly beneath this one
    // contains filtered or unexported fields
}
```

Function represents the parameters, results, and code of a function or method.

If Blocks is nil, this indicates an external function for which no Go source code is available. In this case, FreeVars, Locals, and Params are nil too. Clients performing whole-program analysis must handle external functions specially.

Blocks contains the function's control-flow graph (CFG). Blocks\[0\] is the function entry point; block order is not otherwise semantically significant, though it may affect the readability of the disassembly. To iterate over the blocks in dominance order, use DomPreorder().

Recover is an optional second entry point to which control resumes after a recovered panic. The Recover block may contain only a return statement, preceded by a load of the function's named return parameters, if any.

A nested function (Parent()!=nil) that refers to one or more lexically enclosing local variables ("free variables") has FreeVars. Such functions cannot be called directly but require a value created by MakeClosure which, via its Bindings, supplies values for these parameters.

If the function is a method (Signature.Recv()!= nil) then the first element of Params is the receiver parameter.

A Go package may declare many functions called "init". For each one, Object().Name() returns "init" but Name() returns "init#1", etc, in declaration order.

Pos() returns the declaring ast.FuncLit.Type.Func or the position of the ast.FuncDecl.Name, if the function was explicit in the source. Synthetic wrappers, for which Synthetic!= "", may share the same position as the function they wrap. Syntax.Pos() always returns the position of the declaring "func" token.

When the operand of a range statement is an iterator function, the loop body is transformed into a synthetic anonymous function that is passed as the yield argument in a call to the iterator. In that case, Function.Pos is the position of the "range" token, and Function.Syntax is the ast.RangeStmt.

Synthetic functions, for which Synthetic!= "", are functions that do not appear in the source AST. These include:

- method wrappers,
- thunks,
- bound functions,
- empty functions built from loaded type information,
- yield functions created from range-over-func loops,
- package init functions, and
- instantiations of generic functions.

Synthetic wrapper functions may share the same position as the function they wrap.

Type() returns the function's Signature.

A generic function is a function or method that has uninstantiated type parameters (TypeParams()!= nil). Consider a hypothetical generic method, (\*Map\[K,V\]).Get. It may be instantiated with all non-parameterized types as (\*Map\[string,int\]).Get or with parameterized types as (\*Map\[string,U\]).Get, where U is a type parameter. In both instantiations, Origin() refers to the instantiated generic method, (\*Map\[K,V\]).Get, TypeParams() refers to the parameters \[K,V\] of the generic method. TypeArgs() refers to \[string,U\] or \[string,int\], respectively, and is nil in the generic method.

#### func EnclosingFunction ¶

```
func EnclosingFunction(pkg *Package, path []ast.Node) *Function
```

EnclosingFunction returns the function that contains the syntax node denoted by path.

Syntax associated with package-level variable specifications is enclosed by the package's init() function.

Returns nil if not found; reasons might include:

- the node is not enclosed by any function.
- the node is within an anonymous function (FuncLit) and its SSA function has not been created yet (pkg.Build() has not yet been called).

#### added in v0.18.0

```
func (f *Function) DomPostorder() []*BasicBlock
```

DomPostorder returns a new slice containing the blocks of f in a postorder traversal of the dominator tree. (This is not the same as a postdominance order.)

#### func (\*Function) DomPreorder ¶

```
func (f *Function) DomPreorder() []*BasicBlock
```

DomPreorder returns a new slice containing the blocks of f in a preorder traversal of the dominator tree.

#### func (\*Function) Name ¶

```
func (v *Function) Name() string
```

#### func (\*Function) Object ¶

```
func (v *Function) Object() types.Object
```

#### func (\*Function) Operands ¶

```
func (v *Function) Operands(rands []*Value) []*Value
```

#### added in v0.4.0

```
func (fn *Function) Origin() *Function
```

Origin returns the generic function from which fn was instantiated, or nil if fn is not an instantiation.

#### func (\*Function) Package ¶

```
func (v *Function) Package() *Package
```

#### func (\*Function) Parent ¶

```
func (v *Function) Parent() *Function
```

#### func (\*Function) Pos ¶

```
func (v *Function) Pos() token.Pos
```

#### func (\*Function) Referrers ¶

```
func (v *Function) Referrers() *[]Instruction
```

#### func (\*Function) RelString ¶

```
func (f *Function) RelString(from *types.Package) string
```

RelString returns the full name of this function, qualified by package name, receiver type, etc.

The specific formatting rules are not guaranteed and may change.

Examples:

```
"math.IsNaN"                  // a package-level function
"(*bytes.Buffer).Bytes"       // a declared method or a wrapper
"(*bytes.Buffer).Bytes$thunk" // thunk (func wrapping method; receiver is param 0)
"(*bytes.Buffer).Bytes$bound" // bound (func wrapping method; receiver supplied by closure)
"main.main$1"                 // an anonymous function in main
"main.init#1"                 // a declared init function
"main.init"                   // the synthesized package initializer
```

When these functions are referred to from within the same package (i.e. from == f.Pkg.Object), they are rendered without the package path. For example: "IsNaN", "(\*Buffer).Bytes", etc.

All non-synthetic functions have distinct package-qualified names. (But two methods may have the same name "(T).f" if one is a synthetic wrapper promoting a non-exported method "f" from another package; in that case, the strings are equal but the identifiers "f" are distinct.)

#### func (\*Function) String ¶

```
func (v *Function) String() string
```

#### func (\*Function) Syntax ¶

```
func (f *Function) Syntax() ast.Node
```

Syntax returns the function's syntax (\*ast.Func{Decl,Lit}) if it was produced from syntax or an \*ast.RangeStmt if it is a range-over-func yield function.

#### func (\*Function) Token ¶

```
func (v *Function) Token() token.Token
```

#### func (\*Function) Type ¶

```
func (v *Function) Type() types.Type
```

#### added in v0.4.0

```
func (fn *Function) TypeArgs() []types.Type
```

TypeArgs are the types that TypeParams() were instantiated by to create fn from fn.Origin().

Specifically, the resulting slice behaves like:

```
f                   // []
f[int]              // [int]
T.m                 // []
T.m[int]            // [int]
T[int].m            // [int]
T[int].m[uint]      // [int, uint]
```

Note that receiver type arguments precede other type arguments.

#### added in v0.4.0

```
func (fn *Function) TypeParams() *types.TypeParamList
```

TypeParams are the function's type parameters if generic or the type parameters that were instantiated if fn is an instantiation.

Specifically, the resulting list behaves like:

```
func        f       // []
func        f[P]    // [P]
func (T)    m       // []
func (T)    m[P]    // [P]
func (T[P]) m       // [P]
func (T[P]) m[Q]    // [P (index=0), Q (index=0)]
```

Note that receiver type parameters precede other type parameters. Also, type parameters may have the same index if they come from different source type parameter lists.

#### func (\*Function) ValueForExpr ¶

```
func (f *Function) ValueForExpr(e ast.Expr) (value Value, isAddr bool)
```

ValueForExpr returns the SSA Value that corresponds to non-constant expression e.

It returns nil if no value was found, e.g.

- the expression is not lexically contained within f;
- f was not built with debug information; or
- e is a constant expression. (For efficiency, no debug information is stored for constants. Use go/types.Info.Types\[e\].Value instead.)
- e is a reference to nil or a built-in function.
- the value was optimised away.

If e is an addressable expression used in an lvalue context, value is the address denoted by e, and isAddr is true.

The types of e (or &e, if isAddr) and the result are equal (modulo "untyped" bools resulting from comparisons).

(Tip: to find the ssa.Value given a source position, use astutil.PathEnclosingInterval to locate the ast.Node, then EnclosingFunction to locate the Function, then ValueForExpr to find the ssa.Value.)

#### func (\*Function) WriteTo ¶

```
func (f *Function) WriteTo(w io.Writer) (int64, error)
```

#### type Global ¶

```
type Global struct {
    Pkg *Package
    // contains filtered or unexported fields
}
```

A Global is a named Value holding the address of a package-level variable.

Pos() returns the position of the ast.ValueSpec.Names\[\*\] identifier.

#### func (\*Global) Name ¶

```
func (v *Global) Name() string
```

#### func (\*Global) Object ¶

```
func (v *Global) Object() types.Object
```

#### func (\*Global) Operands ¶

```
func (v *Global) Operands(rands []*Value) []*Value
```

#### func (\*Global) Package ¶

```
func (v *Global) Package() *Package
```

#### func (\*Global) Parent ¶

```
func (v *Global) Parent() *Function
```

#### func (\*Global) Pos ¶

```
func (v *Global) Pos() token.Pos
```

#### func (\*Global) Referrers ¶

```
func (v *Global) Referrers() *[]Instruction
```

#### func (\*Global) RelString ¶

```
func (v *Global) RelString(from *types.Package) string
```

#### func (\*Global) String ¶

```
func (v *Global) String() string
```

#### func (\*Global) Token ¶

```
func (v *Global) Token() token.Token
```

#### func (\*Global) Type ¶

```
func (v *Global) Type() types.Type
```

#### type Go ¶

```
type Go struct {
    Call CallCommon
    // contains filtered or unexported fields
}
```

The Go instruction creates a new goroutine and calls the specified function within it.

See CallCommon for generic function call documentation.

Pos() returns the ast.GoStmt.Go.

Example printed form:

```
go println(t0, t1)
go t3()
go invoke t5.Println(...t6)
```

#### func (\*Go) Block ¶

```
func (v *Go) Block() *BasicBlock
```

#### func (\*Go) Common ¶

```
func (s *Go) Common() *CallCommon
```

#### func (\*Go) Operands ¶

```
func (s *Go) Operands(rands []*Value) []*Value
```

#### func (\*Go) Parent ¶

```
func (v *Go) Parent() *Function
```

#### func (\*Go) Pos ¶

```
func (s *Go) Pos() token.Pos
```

#### func (\*Go) Referrers ¶

```
func (v *Go) Referrers() *[]Instruction
```

#### func (\*Go) String ¶

```
func (s *Go) String() string
```

#### func (\*Go) Value ¶

```
func (s *Go) Value() *Call
```

#### type If ¶

```
type If struct {
    Cond Value
    // contains filtered or unexported fields
}
```

The If instruction transfers control to one of the two successors of its owning block, depending on the boolean Cond: the first if true, the second if false.

An If instruction must be the last instruction of its containing BasicBlock.

Pos() returns NoPos.

Example printed form:

```
if t0 goto done else body
```

#### func (\*If) Block ¶

```
func (v *If) Block() *BasicBlock
```

#### func (\*If) Operands ¶

```
func (s *If) Operands(rands []*Value) []*Value
```

#### func (\*If) Parent ¶

```
func (v *If) Parent() *Function
```

#### func (\*If) Pos ¶

```
func (s *If) Pos() token.Pos
```

#### func (\*If) Referrers ¶

```
func (v *If) Referrers() *[]Instruction
```

#### func (\*If) String ¶

```
func (s *If) String() string
```

#### type Index ¶

```
type Index struct {
    X     Value // array, string or type parameter with types array, *array, slice, or string.
    Index Value // integer index
    // contains filtered or unexported fields
}
```

The Index instruction yields element Index of collection X, an array, string or type parameter containing an array, a string, a pointer to an, array or a slice.

Pos() returns the ast.IndexExpr.Lbrack for the index operation, if explicit in the source.

Example printed form:

```
t2 = t0[t1]
```

#### func (\*Index) Name ¶

```
func (v *Index) Name() string
```

#### func (\*Index) Operands ¶

```
func (v *Index) Operands(rands []*Value) []*Value
```

#### func (\*Index) Pos ¶

```
func (v *Index) Pos() token.Pos
```

#### func (\*Index) Referrers ¶

```
func (v *Index) Referrers() *[]Instruction
```

#### func (\*Index) String ¶

```
func (v *Index) String() string
```

#### func (\*Index) Type ¶

```
func (v *Index) Type() types.Type
```

#### type IndexAddr ¶

```
type IndexAddr struct {
    X     Value // *array, slice or type parameter with types array, *array, or slice.
    Index Value // numeric index
    // contains filtered or unexported fields
}
```

The IndexAddr instruction yields the address of the element at index Index of collection X. Index is an integer expression.

The elements of maps and strings are not addressable; use Lookup (map), Index (string), or MapUpdate instead.

Dynamically, this instruction panics if X evaluates to a nil \*array pointer.

Type() returns a (possibly named) \*types.Pointer.

Pos() returns the ast.IndexExpr.Lbrack for the index operation, if explicit in the source.

Example printed form:

```
t2 = &t0[t1]
```

#### func (\*IndexAddr) Name ¶

```
func (v *IndexAddr) Name() string
```

#### func (\*IndexAddr) Operands ¶

```
func (v *IndexAddr) Operands(rands []*Value) []*Value
```

#### func (\*IndexAddr) Pos ¶

```
func (v *IndexAddr) Pos() token.Pos
```

#### func (\*IndexAddr) Referrers ¶

```
func (v *IndexAddr) Referrers() *[]Instruction
```

#### func (\*IndexAddr) String ¶

```
func (v *IndexAddr) String() string
```

#### func (\*IndexAddr) Type ¶

```
func (v *IndexAddr) Type() types.Type
```

#### type Instruction ¶

```
type Instruction interface {
    // String returns the disassembled form of this value.
    //
    // Examples of Instructions that are Values:
    //       "x + y"     (BinOp)
    //       "len([])"   (Call)
    // Note that the name of the Value is not printed.
    //
    // Examples of Instructions that are not Values:
    //       "return x"  (Return)
    //       "*y = x"    (Store)
    //
    // (The separation Value.Name() from Value.String() is useful
    // for some analyses which distinguish the operation from the
    // value it defines, e.g., 'y = local int' is both an allocation
    // of memory 'local int' and a definition of a pointer y.)
    String() string

    // Parent returns the function to which this instruction
    // belongs.
    Parent() *Function

    // Block returns the basic block to which this instruction
    // belongs.
    Block() *BasicBlock

    // Operands returns the operands of this instruction: the
    // set of Values it references.
    //
    // Specifically, it appends their addresses to rands, a
    // user-provided slice, and returns the resulting slice,
    // permitting avoidance of memory allocation.
    //
    // The operands are appended in undefined order, but the order
    // is consistent for a given Instruction; the addresses are
    // always non-nil but may point to a nil Value.  Clients may
    // store through the pointers, e.g. to effect a value
    // renaming.
    //
    // Value.Referrers is a subset of the inverse of this
    // relation.  (Referrers are not tracked for all types of
    // Values.)
    Operands(rands []*Value) []*Value

    // Pos returns the location of the AST token most closely
    // associated with the operation that gave rise to this
    // instruction, or token.NoPos if it was not explicit in the
    // source.
    //
    // For each ast.Node type, a particular token is designated as
    // the closest location for the expression, e.g. the Go token
    // for an *ast.GoStmt.  This permits a compact but approximate
    // mapping from Instructions to source positions for use in
    // diagnostic messages, for example.
    //
    // (Do not use this position to determine which Instruction
    // corresponds to an ast.Expr; see the notes for Value.Pos.
    // This position may be used to determine which non-Value
    // Instruction corresponds to some ast.Stmts, but not all: If
    // and Jump instructions have no Pos(), for example.)
    Pos() token.Pos
    // contains filtered or unexported methods
}
```

An Instruction is an SSA instruction that computes a new Value or has some effect.

An Instruction that defines a value (e.g. BinOp) also implements the Value interface; an Instruction that only has an effect (e.g. Store) does not.

#### type Jump ¶

```
type Jump struct {
    // contains filtered or unexported fields
}
```

The Jump instruction transfers control to the sole successor of its owning block.

A Jump must be the last instruction of its containing BasicBlock.

Pos() returns NoPos.

Example printed form:

```
jump done
```

#### func (\*Jump) Block ¶

```
func (v *Jump) Block() *BasicBlock
```

#### func (\*Jump) Operands ¶

```
func (*Jump) Operands(rands []*Value) []*Value
```

#### func (\*Jump) Parent ¶

```
func (v *Jump) Parent() *Function
```

#### func (\*Jump) Pos ¶

```
func (s *Jump) Pos() token.Pos
```

#### func (\*Jump) Referrers ¶

```
func (v *Jump) Referrers() *[]Instruction
```

#### func (\*Jump) String ¶

```
func (s *Jump) String() string
```

#### type Lookup ¶

```
type Lookup struct {
    X       Value // map
    Index   Value // key-typed index
    CommaOk bool  // return a value,ok pair
    // contains filtered or unexported fields
}
```

The Lookup instruction yields element Index of collection map X. Index is the appropriate key type.

If CommaOk, the result is a 2-tuple of the value above and a boolean indicating the result of a map membership test for the key. The components of the tuple are accessed using Extract.

Pos() returns the ast.IndexExpr.Lbrack, if explicit in the source.

Example printed form:

```
t2 = t0[t1]
t5 = t3[t4],ok
```

#### func (\*Lookup) Name ¶

```
func (v *Lookup) Name() string
```

#### func (\*Lookup) Operands ¶

```
func (v *Lookup) Operands(rands []*Value) []*Value
```

#### func (\*Lookup) Pos ¶

```
func (v *Lookup) Pos() token.Pos
```

#### func (\*Lookup) Referrers ¶

```
func (v *Lookup) Referrers() *[]Instruction
```

#### func (\*Lookup) String ¶

```
func (v *Lookup) String() string
```

#### func (\*Lookup) Type ¶

```
func (v *Lookup) Type() types.Type
```

#### type MakeChan ¶

```
type MakeChan struct {
    Size Value // int; size of buffer; zero => synchronous.
    // contains filtered or unexported fields
}
```

The MakeChan instruction creates a new channel object and yields a value of kind chan.

Type() returns a (possibly named) \*types.Chan.

Pos() returns the ast.CallExpr.Lparen for the make(chan) that created it.

Example printed form:

```
t0 = make chan int 0
t0 = make IntChan 0
```

#### func (\*MakeChan) Name ¶

```
func (v *MakeChan) Name() string
```

#### func (\*MakeChan) Operands ¶

```
func (v *MakeChan) Operands(rands []*Value) []*Value
```

#### func (\*MakeChan) Pos ¶

```
func (v *MakeChan) Pos() token.Pos
```

#### func (\*MakeChan) Referrers ¶

```
func (v *MakeChan) Referrers() *[]Instruction
```

#### func (\*MakeChan) String ¶

```
func (v *MakeChan) String() string
```

#### func (\*MakeChan) Type ¶

```
func (v *MakeChan) Type() types.Type
```

#### type MakeClosure ¶

```
type MakeClosure struct {
    Fn       Value   // always a *Function
    Bindings []Value // values for each free variable in Fn.FreeVars
    // contains filtered or unexported fields
}
```

The MakeClosure instruction yields a closure value whose code is Fn and whose free variables' values are supplied by Bindings.

Type() returns a (possibly named) \*types.Signature.

Pos() returns the ast.FuncLit.Type.Func for a function literal closure or the ast.SelectorExpr.Sel for a bound method closure.

Example printed form:

```
t0 = make closure anon@1.2 [x y z]
t1 = make closure bound$(main.I).add [i]
```

#### func (\*MakeClosure) Name ¶

```
func (v *MakeClosure) Name() string
```

#### func (\*MakeClosure) Operands ¶

```
func (v *MakeClosure) Operands(rands []*Value) []*Value
```

#### func (\*MakeClosure) Pos ¶

```
func (v *MakeClosure) Pos() token.Pos
```

#### func (\*MakeClosure) Referrers ¶

```
func (v *MakeClosure) Referrers() *[]Instruction
```

#### func (\*MakeClosure) String ¶

```
func (v *MakeClosure) String() string
```

#### func (\*MakeClosure) Type ¶

```
func (v *MakeClosure) Type() types.Type
```

#### type MakeInterface ¶

```
type MakeInterface struct {
    X Value
    // contains filtered or unexported fields
}
```

MakeInterface constructs an instance of an interface type from a value of a concrete type.

Use Program.MethodSets.MethodSet(X.Type()) to find the method-set of X, and Program.MethodValue(m) to find the implementation of a method.

To construct the zero value of an interface type T, use:

```
NewConst(constant.MakeNil(), T, pos)
```

Pos() returns the ast.CallExpr.Lparen, if the instruction arose from an explicit conversion in the source.

Example printed form:

```
t1 = make interface{} <- int (42:int)
t2 = make Stringer <- t0
```

#### func (\*MakeInterface) Name ¶

```
func (v *MakeInterface) Name() string
```

#### func (\*MakeInterface) Operands ¶

```
func (v *MakeInterface) Operands(rands []*Value) []*Value
```

#### func (\*MakeInterface) Pos ¶

```
func (v *MakeInterface) Pos() token.Pos
```

#### func (\*MakeInterface) Referrers ¶

```
func (v *MakeInterface) Referrers() *[]Instruction
```

#### func (\*MakeInterface) String ¶

```
func (v *MakeInterface) String() string
```

#### func (\*MakeInterface) Type ¶

```
func (v *MakeInterface) Type() types.Type
```

#### type MakeMap ¶

```
type MakeMap struct {
    Reserve Value // initial space reservation; nil => default
    // contains filtered or unexported fields
}
```

The MakeMap instruction creates a new hash-table-based map object and yields a value of kind map.

Type() returns a (possibly named) \*types.Map.

Pos() returns the ast.CallExpr.Lparen, if created by make(map), or the ast.CompositeLit.Lbrack if created by a literal.

Example printed form:

```
t1 = make map[string]int t0
t1 = make StringIntMap t0
```

#### func (\*MakeMap) Name ¶

```
func (v *MakeMap) Name() string
```

#### func (\*MakeMap) Operands ¶

```
func (v *MakeMap) Operands(rands []*Value) []*Value
```

#### func (\*MakeMap) Pos ¶

```
func (v *MakeMap) Pos() token.Pos
```

#### func (\*MakeMap) Referrers ¶

```
func (v *MakeMap) Referrers() *[]Instruction
```

#### func (\*MakeMap) String ¶

```
func (v *MakeMap) String() string
```

#### func (\*MakeMap) Type ¶

```
func (v *MakeMap) Type() types.Type
```

#### type MakeSlice ¶

```
type MakeSlice struct {
    Len Value
    Cap Value
    // contains filtered or unexported fields
}
```

The MakeSlice instruction yields a slice of length Len backed by a newly allocated array of length Cap.

Both Len and Cap must be non-nil Values of integer type.

(Alloc(types.Array) followed by Slice will not suffice because Alloc can only create arrays of constant length.)

Type() returns a (possibly named) \*types.Slice.

Pos() returns the ast.CallExpr.Lparen for the make(\[\]T) that created it.

Example printed form:

```
t1 = make []string 1:int t0
t1 = make StringSlice 1:int t0
```

#### func (\*MakeSlice) Name ¶

```
func (v *MakeSlice) Name() string
```

#### func (\*MakeSlice) Operands ¶

```
func (v *MakeSlice) Operands(rands []*Value) []*Value
```

#### func (\*MakeSlice) Pos ¶

```
func (v *MakeSlice) Pos() token.Pos
```

#### func (\*MakeSlice) Referrers ¶

```
func (v *MakeSlice) Referrers() *[]Instruction
```

#### func (\*MakeSlice) String ¶

```
func (v *MakeSlice) String() string
```

#### func (\*MakeSlice) Type ¶

```
func (v *MakeSlice) Type() types.Type
```

#### type MapUpdate ¶

```
type MapUpdate struct {
    Map   Value
    Key   Value
    Value Value
    // contains filtered or unexported fields
}
```

The MapUpdate instruction updates the association of Map\[Key\] to Value.

Pos() returns the ast.KeyValueExpr.Colon or ast.IndexExpr.Lbrack, if explicit in the source.

Example printed form:

```
t0[t1] = t2
```

#### func (\*MapUpdate) Block ¶

```
func (v *MapUpdate) Block() *BasicBlock
```

#### func (\*MapUpdate) Operands ¶

```
func (v *MapUpdate) Operands(rands []*Value) []*Value
```

#### func (\*MapUpdate) Parent ¶

```
func (v *MapUpdate) Parent() *Function
```

#### func (\*MapUpdate) Pos ¶

```
func (s *MapUpdate) Pos() token.Pos
```

#### func (\*MapUpdate) Referrers ¶

```
func (v *MapUpdate) Referrers() *[]Instruction
```

#### func (\*MapUpdate) String ¶

```
func (s *MapUpdate) String() string
```

#### type Member ¶

```
type Member interface {
    Name() string                    // declared name of the package member
    String() string                  // package-qualified name of the package member
    RelString(*types.Package) string // like String, but relative refs are unqualified
    Object() types.Object            // typechecker's object for this member, if any
    Pos() token.Pos                  // position of member's declaration, if known
    Type() types.Type                // type of the package member
    Token() token.Token              // token.{VAR,FUNC,CONST,TYPE}
    Package() *Package               // the containing package
}
```

A Member is a member of a Go package, implemented by \*NamedConst, \*Global, \*Function, or \*Type; they are created by package-level const, var, func and type declarations respectively.

#### added in v0.6.0

```
type MultiConvert struct {
    X Value
    // contains filtered or unexported fields
}
```

The MultiConvert instruction yields the conversion of value X to type Type(). Either X.Type() or Type() must be a type parameter. Each type in the type set of X.Type() can be converted to each type in the type set of Type().

See the documentation for Convert, ChangeType, and SliceToArrayPointer for the conversions that are permitted. Additionally conversions of slices to arrays are permitted.

This operation can fail dynamically (see SliceToArrayPointer).

Pos() returns the ast.CallExpr.Lparen, if the instruction arose from an explicit conversion in the source.

Example printed form:

```
t1 = multiconvert D <- S (t0) [*[2]rune <- []rune | string <- []rune]
```

#### added in v0.6.0

```
func (v *MultiConvert) Name() string
```

#### added in v0.6.0

```
func (v *MultiConvert) Operands(rands []*Value) []*Value
```

#### added in v0.6.0

```
func (v *MultiConvert) Pos() token.Pos
```

#### added in v0.6.0

```
func (v *MultiConvert) Referrers() *[]Instruction
```

#### added in v0.6.0

```
func (v *MultiConvert) String() string
```

#### added in v0.6.0

```
func (v *MultiConvert) Type() types.Type
```

#### type NamedConst ¶

```
type NamedConst struct {
    Value *Const
    // contains filtered or unexported fields
}
```

A NamedConst is a Member of a Package representing a package-level named constant.

Pos() returns the position of the declaring ast.ValueSpec.Names\[\*\] identifier.

NB: a NamedConst is not a Value; it contains a constant Value, which it augments with the name and position of its 'const' declaration.

#### func (\*NamedConst) Name ¶

```
func (c *NamedConst) Name() string
```

#### func (\*NamedConst) Object ¶

```
func (c *NamedConst) Object() types.Object
```

#### func (\*NamedConst) Package ¶

```
func (c *NamedConst) Package() *Package
```

#### func (\*NamedConst) Pos ¶

```
func (c *NamedConst) Pos() token.Pos
```

#### func (\*NamedConst) RelString ¶

```
func (c *NamedConst) RelString(from *types.Package) string
```

#### func (\*NamedConst) String ¶

```
func (c *NamedConst) String() string
```

#### func (\*NamedConst) Token ¶

```
func (c *NamedConst) Token() token.Token
```

#### func (\*NamedConst) Type ¶

```
func (c *NamedConst) Type() types.Type
```

#### type Next ¶

```
type Next struct {
    Iter     Value
    IsString bool // true => string iterator; false => map iterator.
    // contains filtered or unexported fields
}
```

The Next instruction reads and advances the (map or string) iterator Iter and returns a 3-tuple value (ok, k, v). If the iterator is not exhausted, ok is true and k and v are the next elements of the domain and range, respectively. Otherwise ok is false and k and v are undefined.

Components of the tuple are accessed using Extract.

The IsString field distinguishes iterators over strings from those over maps, as the Type() alone is insufficient: consider map\[int\]rune.

Type() returns a \*types.Tuple for the triple (ok, k, v). The types of k and/or v may be types.Invalid.

Example printed form:

```
t1 = next t0
```

#### func (\*Next) Name ¶

```
func (v *Next) Name() string
```

#### func (\*Next) Operands ¶

```
func (v *Next) Operands(rands []*Value) []*Value
```

#### func (\*Next) Pos ¶

```
func (v *Next) Pos() token.Pos
```

#### func (\*Next) Referrers ¶

```
func (v *Next) Referrers() *[]Instruction
```

#### func (\*Next) String ¶

```
func (v *Next) String() string
```

#### func (\*Next) Type ¶

```
func (v *Next) Type() types.Type
```

#### type Node ¶

```
type Node interface {
    // Common methods:
    String() string
    Pos() token.Pos
    Parent() *Function

    // Partial methods:
    Operands(rands []*Value) []*Value // nil for non-Instructions
    Referrers() *[]Instruction        // nil for non-Values
}
```

A Node is a node in the SSA value graph. Every concrete type that implements Node is also either a Value, an Instruction, or both.

Node contains the methods common to Value and Instruction, plus the Operands and Referrers methods generalized to return nil for non-Instructions and non-Values, respectively.

Node is provided to simplify SSA graph algorithms. Clients should use the more specific and informative Value or Instruction interfaces where appropriate.

#### type Package ¶

```
type Package struct {
    Prog    *Program          // the owning program
    Pkg     *types.Package    // the corresponding go/types.Package
    Members map[string]Member // all package members keyed by name (incl. init and init#%d)
    // contains filtered or unexported fields
}
```

A Package is a single analyzed Go package containing Members for all package-level functions, variables, constants and types it declares. These may be accessed directly via Members, or via the type-specific accessor methods Func, Type, Var and Const.

Members also contains entries for "init" (the synthetic package initializer) and "init#%d", the nth declared init function, and unspecified other things too.

#### func (\*Package) Build ¶

```
func (p *Package) Build()
```

Build builds SSA code for all functions and vars in package p.

CreatePackage must have been called for all of p's direct imports (and hence its direct imports must have been error-free). It is not necessary to call CreatePackage for indirect dependencies. Functions will be created for all necessary methods in those packages on demand.

Build is idempotent and thread-safe.

#### func (\*Package) Const ¶

```
func (p *Package) Const(name string) (c *NamedConst)
```

Const returns the package-level constant of the specified name, or nil if not found.

#### func (\*Package) Func ¶

```
func (p *Package) Func(name string) (f *Function)
```

Func returns the package-level function of the specified name, or nil if not found.

#### func (\*Package) SetDebugMode ¶

```
func (pkg *Package) SetDebugMode(debug bool)
```

SetDebugMode sets the debug mode for package pkg. If true, all its functions will include full debug info. This greatly increases the size of the instruction stream, and causes Functions to depend upon the ASTs, potentially keeping them live in memory for longer.

#### func (\*Package) String ¶

```
func (p *Package) String() string
```

#### func (\*Package) Type ¶

```
func (p *Package) Type(name string) (t *Type)
```

Type returns the package-level type of the specified name, or nil if not found.

#### func (\*Package) Var ¶

```
func (p *Package) Var(name string) (g *Global)
```

Var returns the package-level variable of the specified name, or nil if not found.

#### func (\*Package) WriteTo ¶

```
func (p *Package) WriteTo(w io.Writer) (int64, error)
```

#### type Panic ¶

```
type Panic struct {
    X Value // an interface{}
    // contains filtered or unexported fields
}
```

The Panic instruction initiates a panic with value X.

A Panic instruction must be the last instruction of its containing BasicBlock, which must have no successors.

NB: 'go panic(x)' and 'defer panic(x)' do not use this instruction; they are treated as calls to a built-in function.

Pos() returns the ast.CallExpr.Lparen if this panic was explicit in the source.

Example printed form:

```
panic t0
```

#### func (\*Panic) Block ¶

```
func (v *Panic) Block() *BasicBlock
```

#### func (\*Panic) Operands ¶

```
func (s *Panic) Operands(rands []*Value) []*Value
```

#### func (\*Panic) Parent ¶

```
func (v *Panic) Parent() *Function
```

#### func (\*Panic) Pos ¶

```
func (s *Panic) Pos() token.Pos
```

#### func (\*Panic) Referrers ¶

```
func (v *Panic) Referrers() *[]Instruction
```

#### func (\*Panic) String ¶

```
func (s *Panic) String() string
```

#### type Parameter ¶

```
type Parameter struct {
    // contains filtered or unexported fields
}
```

A Parameter represents an input parameter of a function.

#### func (\*Parameter) Name ¶

```
func (v *Parameter) Name() string
```

#### func (\*Parameter) Object ¶

```
func (v *Parameter) Object() types.Object
```

#### func (\*Parameter) Operands ¶

```
func (v *Parameter) Operands(rands []*Value) []*Value
```

#### func (\*Parameter) Parent ¶

```
func (v *Parameter) Parent() *Function
```

#### func (\*Parameter) Pos ¶

```
func (v *Parameter) Pos() token.Pos
```

#### func (\*Parameter) Referrers ¶

```
func (v *Parameter) Referrers() *[]Instruction
```

#### func (\*Parameter) String ¶

```
func (v *Parameter) String() string
```

#### func (\*Parameter) Type ¶

```
func (v *Parameter) Type() types.Type
```

#### type Phi ¶

```
type Phi struct {
    Comment string  // a hint as to its purpose
    Edges   []Value // Edges[i] is value for Block().Preds[i]
    // contains filtered or unexported fields
}
```

The Phi instruction represents an SSA φ-node, which combines values that differ across incoming control-flow edges and yields a new value. Within a block, all φ-nodes must appear before all non-φ nodes.

Pos() returns the position of the && or || for short-circuit control-flow joins, or that of the \*Alloc for φ-nodes inserted during SSA renaming.

Example printed form:

```
t2 = phi [0: t0, 1: t1]
```

#### func (\*Phi) Name ¶

```
func (v *Phi) Name() string
```

#### func (\*Phi) Operands ¶

```
func (v *Phi) Operands(rands []*Value) []*Value
```

#### func (\*Phi) Pos ¶

```
func (v *Phi) Pos() token.Pos
```

#### func (\*Phi) Referrers ¶

```
func (v *Phi) Referrers() *[]Instruction
```

#### func (\*Phi) String ¶

```
func (v *Phi) String() string
```

#### func (\*Phi) Type ¶

```
func (v *Phi) Type() types.Type
```

#### type Program ¶

```
type Program struct {
    Fset *token.FileSet // position information for the files of this Program

    MethodSets typeutil.MethodSetCache // cache of type-checker's method-sets
    // contains filtered or unexported fields
}
```

A Program is a partial or complete Go program converted to SSA form.

#### func NewProgram ¶

```
func NewProgram(fset *token.FileSet, mode BuilderMode) *Program
```

NewProgram returns a new SSA Program.

mode controls diagnostics and checking during SSA construction.

To construct an SSA program:

- Call NewProgram to create an empty Program.
- Call CreatePackage providing typed syntax for each package you want to build, and call it with types but not syntax for each of those package's direct dependencies.
- Call [Package.Build](#Package.Build) on each syntax package you wish to build, or [Program.Build](#Program.Build) to build all of them.

See the Example tests for simple examples.

#### func (\*Program) AllPackages ¶

```
func (prog *Program) AllPackages() []*Package
```

AllPackages returns a new slice containing all packages created by prog.CreatePackage in unspecified order.

#### func (\*Program) Build ¶

```
func (prog *Program) Build()
```

Build calls Package.Build for each package in prog. Building occurs in parallel unless the BuildSerially mode flag was set.

Build is intended for whole-program analysis; a typical compiler need only build a single package.

Build is idempotent and thread-safe.

#### func (\*Program) ConstValue ¶

```
func (prog *Program) ConstValue(obj *types.Const) *Const
```

ConstValue returns the SSA constant denoted by the specified const symbol.

#### func (\*Program) CreatePackage ¶

```
func (prog *Program) CreatePackage(pkg *types.Package, files []*ast.File, info *types.Info, importable bool) *Package
```

CreatePackage creates and returns an SSA Package from the specified type-checked, error-free file ASTs, and populates its Members mapping.

importable determines whether this package should be returned by a subsequent call to ImportedPackage(pkg.Path()).

The real work of building SSA form for each function is not done until a subsequent call to Package.Build.

#### func (\*Program) FuncValue ¶

```
func (prog *Program) FuncValue(obj *types.Func) *Function
```

FuncValue returns the SSA function or (non-interface) method denoted by the specified func symbol. It returns nil if the symbol denotes an interface method, or belongs to a package that was not created by prog.CreatePackage.

#### func (\*Program) ImportedPackage ¶

```
func (prog *Program) ImportedPackage(path string) *Package
```

ImportedPackage returns the importable Package whose PkgPath is path, or nil if no such Package has been created.

A parameter to CreatePackage determines whether a package should be considered importable. For example, no import declaration can resolve to the ad-hoc main package created by 'go build foo.go'.

TODO(adonovan): rethink this function and the "importable" concept; most packages are importable. This function assumes that all types.Package.Path values are unique within the ssa.Program, which is false---yet this function remains very convenient. Clients should use (\*Program).Package instead where possible. SSA doesn't really need a string-keyed map of packages.

Furthermore, the graph of packages may contain multiple variants (e.g. "p" vs "p as compiled for q.test"), and each has a different view of its dependencies.

#### func (\*Program) LookupMethod ¶

```
func (prog *Program) LookupMethod(T types.Type, pkg *types.Package, name string) *Function
```

LookupMethod returns the implementation of the method of type T identified by (pkg, name). It returns nil if the method exists but is an interface method or generic method, and panics if T has no such method.

#### func (\*Program) MethodValue ¶

```
func (prog *Program) MethodValue(sel *types.Selection) *Function
```

MethodValue returns the Function implementing method sel, building wrapper methods on demand. It returns nil if sel denotes an interface or generic method.

Precondition: sel.Kind() == MethodVal.

Thread-safe.

Acquires prog.methodsMu.

#### func (\*Program) NewFunction ¶

```
func (prog *Program) NewFunction(name string, sig *types.Signature, provenance string) *Function
```

NewFunction returns a new synthetic Function instance belonging to prog, with its name and signature fields set as specified.

The caller is responsible for initializing the remaining fields of the function object, e.g. Pkg, Params, Blocks.

It is practically impossible for clients to construct well-formed SSA functions/packages/programs directly, so we assume this is the job of the Builder alone. NewFunction exists to provide clients a little flexibility. For example, analysis tools may wish to construct fake Functions for the root of the callgraph, a fake "reflect" package, etc.

TODO(adonovan): think harder about the API here.

#### func (\*Program) Package ¶

```
func (prog *Program) Package(pkg *types.Package) *Package
```

Package returns the SSA Package corresponding to the specified type-checker package. It returns nil if no such Package was created by a prior call to prog.CreatePackage.

#### func (\*Program) RuntimeTypes ¶

```
func (prog *Program) RuntimeTypes() []types.Type
```

RuntimeTypes returns a new unordered slice containing all types in the program for which a runtime type is required.

A runtime type is required for any non-parameterized, non-interface type that is converted to an interface, or for any type (including interface types) derivable from one through reflection.

The methods of such types may be reachable through reflection or interface calls even if they are never called directly.

Thread-safe.

Acquires prog.makeInterfaceTypesMu.

#### added in v0.41.0

```
func (prog *Program) SetNoReturn(noReturn func(*types.Func) bool)
```

SetNoReturn sets the predicate used when building the ssa.Program prog that reports whether a given function cannot return. This may be used to prune spurious control flow edges after (e.g.) log.Fatal, improving the precision of analyses.

A typical implementation is the \[ctrlflow.CFGs.NoReturn\] method from [golang.org/x/tools/go/analysis/passes/ctrlflow](https://pkg.go.dev/golang.org/x/tools@v0.48.0/go/analysis/passes/ctrlflow).

#### func (\*Program) VarValue ¶

```
func (prog *Program) VarValue(obj *types.Var, pkg *Package, ref []ast.Node) (value Value, isAddr bool)
```

VarValue returns the SSA Value that corresponds to a specific identifier denoting the specified var symbol.

VarValue returns nil if a local variable was not found, perhaps because its package was not built, the debug information was not requested during SSA construction, or the value was optimized away.

ref is the path to an ast.Ident (e.g. from PathEnclosingInterval), and that ident must resolve to obj.

pkg is the package enclosing the reference. (A reference to a var always occurs within a function, so we need to know where to find it.)

If the identifier is a field selector and its base expression is non-addressable, then VarValue returns the value of that field. For example:

```
func f() struct {x int}
f().x  // VarValue(x) returns a *Field instruction of type int
```

All other identifiers denote addressable locations (variables). For them, VarValue may return either the variable's address or its value, even when the expression is evaluated only for its value; the situation is reported by isAddr, the second component of the result.

If!isAddr, the returned value is the one associated with the specific identifier. For example,

```
var x int    // VarValue(x) returns Const 0 here
x = 1        // VarValue(x) returns Const 1 here
```

It is not specified whether the value or the address is returned in any particular case, as it may depend upon optimizations performed during SSA code generation, such as registerization, constant folding, avoidance of materialization of subexpressions, etc.

#### type Range ¶

```
type Range struct {
    X Value // string or map
    // contains filtered or unexported fields
}
```

The Range instruction yields an iterator over the domain and range of X, which must be a string or map.

Elements are accessed via Next.

Type() returns an opaque and degenerate "rangeIter" type.

Pos() returns the ast.RangeStmt.For.

Example printed form:

```
t0 = range "hello":string
```

#### func (\*Range) Name ¶

```
func (v *Range) Name() string
```

#### func (\*Range) Operands ¶

```
func (v *Range) Operands(rands []*Value) []*Value
```

#### func (\*Range) Pos ¶

```
func (v *Range) Pos() token.Pos
```

#### func (\*Range) Referrers ¶

```
func (v *Range) Referrers() *[]Instruction
```

#### func (\*Range) String ¶

```
func (v *Range) String() string
```

#### func (\*Range) Type ¶

```
func (v *Range) Type() types.Type
```

#### type Return ¶

```
type Return struct {
    Results []Value
    // contains filtered or unexported fields
}
```

The Return instruction returns values and control back to the calling function.

len(Results) is always equal to the number of results in the function's signature.

If len(Results) > 1, Return returns a tuple value with the specified components which the caller must access using Extract instructions.

There is no instruction to return a ready-made tuple like those returned by a "value,ok"-mode TypeAssert, Lookup or UnOp(ARROW) or a tail-call to a function with multiple result parameters.

Return must be the last instruction of its containing BasicBlock. Such a block has no successors.

Pos() returns the ast.ReturnStmt.Return, if explicit in the source.

Example printed form:

```
return
return nil:I, 2:int
```

#### func (\*Return) Block ¶

```
func (v *Return) Block() *BasicBlock
```

#### func (\*Return) Operands ¶

```
func (s *Return) Operands(rands []*Value) []*Value
```

#### func (\*Return) Parent ¶

```
func (v *Return) Parent() *Function
```

#### func (\*Return) Pos ¶

```
func (s *Return) Pos() token.Pos
```

#### func (\*Return) Referrers ¶

```
func (v *Return) Referrers() *[]Instruction
```

#### func (\*Return) String ¶

```
func (s *Return) String() string
```

#### type RunDefers ¶

```
type RunDefers struct {
    // contains filtered or unexported fields
}
```

The RunDefers instruction pops and invokes the entire stack of procedure calls pushed by Defer instructions in this function.

It is legal to encounter multiple 'rundefers' instructions in a single control-flow path through a function; this is useful in the combined init() function, for example.

Pos() returns NoPos.

Example printed form:

```
rundefers
```

#### func (\*RunDefers) Block ¶

```
func (v *RunDefers) Block() *BasicBlock
```

#### func (\*RunDefers) Operands ¶

```
func (*RunDefers) Operands(rands []*Value) []*Value
```

#### func (\*RunDefers) Parent ¶

```
func (v *RunDefers) Parent() *Function
```

#### func (\*RunDefers) Pos ¶

```
func (s *RunDefers) Pos() token.Pos
```

#### func (\*RunDefers) Referrers ¶

```
func (v *RunDefers) Referrers() *[]Instruction
```

#### func (\*RunDefers) String ¶

```
func (*RunDefers) String() string
```

#### type Select ¶

```
type Select struct {
    States   []*SelectState
    Blocking bool
    // contains filtered or unexported fields
}
```

The Select instruction tests whether (or blocks until) one of the specified sent or received states is entered.

Let n be the number of States for which Dir==RECV and T\_i (0<=i<n) be the element type of each such state's Chan. Select returns an n+2-tuple

```
(index int, recvOk bool, r_0 T_0, ... r_n-1 T_n-1)
```

The tuple's components, described below, must be accessed via the Extract instruction.

If Blocking, select waits until exactly one state holds, i.e. a channel becomes ready for the designated operation of sending or receiving; select chooses one among the ready states pseudorandomly, performs the send or receive operation, and sets 'index' to the index of the chosen channel.

If!Blocking, select doesn't block if no states hold; instead it returns immediately with index equal to -1.

If the chosen channel was used for a receive, the r\_i component is set to the received value, where i is the index of that state among all n receive states; otherwise r\_i has the zero value of type T\_i. Note that the receive index i is not the same as the state index index.

The second component of the triple, recvOk, is a boolean whose value is true iff the selected operation was a receive and the receive successfully yielded a value.

Pos() returns the ast.SelectStmt.Select.

Example printed form:

```
t3 = select nonblocking [<-t0, t1<-t2]
t4 = select blocking []
```

#### func (\*Select) Name ¶

```
func (v *Select) Name() string
```

#### func (\*Select) Operands ¶

```
func (v *Select) Operands(rands []*Value) []*Value
```

#### func (\*Select) Pos ¶

```
func (v *Select) Pos() token.Pos
```

#### func (\*Select) Referrers ¶

```
func (v *Select) Referrers() *[]Instruction
```

#### func (\*Select) String ¶

```
func (s *Select) String() string
```

#### func (\*Select) Type ¶

```
func (v *Select) Type() types.Type
```

#### type SelectState ¶

```
type SelectState struct {
    Dir       types.ChanDir // direction of case (SendOnly or RecvOnly)
    Chan      Value         // channel to use (for send or receive)
    Send      Value         // value to send (for send)
    Pos       token.Pos     // position of token.ARROW
    DebugNode ast.Node      // ast.SendStmt or ast.UnaryExpr(<-) [debug mode]
}
```

SelectState is a helper for Select. It represents one goal state and its corresponding communication.

#### type Send ¶

```
type Send struct {
    Chan, X Value
    // contains filtered or unexported fields
}
```

The Send instruction sends X on channel Chan.

Pos() returns the ast.SendStmt.Arrow, if explicit in the source.

Example printed form:

```
send t0 <- t1
```

#### func (\*Send) Block ¶

```
func (v *Send) Block() *BasicBlock
```

#### func (\*Send) Operands ¶

```
func (s *Send) Operands(rands []*Value) []*Value
```

#### func (\*Send) Parent ¶

```
func (v *Send) Parent() *Function
```

#### func (\*Send) Pos ¶

```
func (s *Send) Pos() token.Pos
```

#### func (\*Send) Referrers ¶

```
func (v *Send) Referrers() *[]Instruction
```

#### func (\*Send) String ¶

```
func (s *Send) String() string
```

#### type Slice ¶

```
type Slice struct {
    X              Value // slice, string, or *array
    Low, High, Max Value // each may be nil
    // contains filtered or unexported fields
}
```

The Slice instruction yields a slice of an existing string, slice or \*array X between optional integer bounds Low and High.

Dynamically, this instruction panics if X evaluates to a nil \*array pointer.

Type() returns string if the type of X was string, otherwise a \*types.Slice with the same element type as X.

Pos() returns the ast.SliceExpr.Lbrack if created by a x\[:\] slice operation, the ast.CompositeLit.Lbrace if created by a literal, or NoPos if not explicit in the source (e.g. a variadic argument slice).

Example printed form:

```
t1 = slice t0[1:]
```

#### func (\*Slice) Name ¶

```
func (v *Slice) Name() string
```

#### func (\*Slice) Operands ¶

```
func (v *Slice) Operands(rands []*Value) []*Value
```

#### func (\*Slice) Pos ¶

```
func (v *Slice) Pos() token.Pos
```

#### func (\*Slice) Referrers ¶

```
func (v *Slice) Referrers() *[]Instruction
```

#### func (\*Slice) String ¶

```
func (v *Slice) String() string
```

#### func (\*Slice) Type ¶

```
func (v *Slice) Type() types.Type
```

#### added in v0.1.6

```
type SliceToArrayPointer struct {
    X Value
    // contains filtered or unexported fields
}
```

The SliceToArrayPointer instruction yields the conversion of slice X to array pointer.

Pos() returns the ast.CallExpr.Lparen, if the instruction arose from an explicit conversion in the source.

Conversion may to be to or from a type parameter. All types in the type set of X.Type() must be a slice types that can be converted to all types in the type set of Type() which must all be pointer to array types.

This operation can fail dynamically if the length of the slice is less than the length of the array.

Example printed form:

```
t1 = slice to array pointer *[4]byte <- []byte (t0)
```

#### added in v0.1.6

```
func (v *SliceToArrayPointer) Name() string
```

#### added in v0.1.6

```
func (v *SliceToArrayPointer) Operands(rands []*Value) []*Value
```

#### added in v0.1.6

```
func (v *SliceToArrayPointer) Pos() token.Pos
```

#### added in v0.1.6

```
func (v *SliceToArrayPointer) Referrers() *[]Instruction
```

#### added in v0.1.6

```
func (v *SliceToArrayPointer) String() string
```

#### added in v0.1.6

```
func (v *SliceToArrayPointer) Type() types.Type
```

#### type Store ¶

```
type Store struct {
    Addr Value
    Val  Value
    // contains filtered or unexported fields
}
```

The Store instruction stores Val at address Addr. Stores can be of arbitrary types.

Pos() returns the position of the source-level construct most closely associated with the memory store operation. Since implicit memory stores are numerous and varied and depend upon implementation choices, the details are not specified.

Example printed form:

```
*x = y
```

#### func (\*Store) Block ¶

```
func (v *Store) Block() *BasicBlock
```

#### func (\*Store) Operands ¶

```
func (s *Store) Operands(rands []*Value) []*Value
```

#### func (\*Store) Parent ¶

```
func (v *Store) Parent() *Function
```

#### func (\*Store) Pos ¶

```
func (s *Store) Pos() token.Pos
```

#### func (\*Store) Referrers ¶

```
func (v *Store) Referrers() *[]Instruction
```

#### func (\*Store) String ¶

```
func (s *Store) String() string
```

#### type Type ¶

```
type Type struct {
    // contains filtered or unexported fields
}
```

A Type is a Member of a Package representing a package-level named type.

#### func (\*Type) Name ¶

```
func (t *Type) Name() string
```

#### func (\*Type) Object ¶

```
func (t *Type) Object() types.Object
```

#### func (\*Type) Package ¶

```
func (t *Type) Package() *Package
```

#### func (\*Type) Pos ¶

```
func (t *Type) Pos() token.Pos
```

#### func (\*Type) RelString ¶

```
func (t *Type) RelString(from *types.Package) string
```

#### func (\*Type) String ¶

```
func (t *Type) String() string
```

#### func (\*Type) Token ¶

```
func (t *Type) Token() token.Token
```

#### func (\*Type) Type ¶

```
func (t *Type) Type() types.Type
```

#### type TypeAssert ¶

```
type TypeAssert struct {
    X            Value
    AssertedType types.Type
    CommaOk      bool
    // contains filtered or unexported fields
}
```

The TypeAssert instruction tests whether interface value X has type AssertedType.

If!CommaOk, on success it returns v, the result of the conversion (defined below); on failure it panics.

If CommaOk: on success it returns a pair (v, true) where v is the result of the conversion; on failure it returns (z, false) where z is AssertedType's zero value. The components of the pair must be accessed using the Extract instruction.

If Underlying: tests whether interface value X has the underlying type AssertedType.

If AssertedType is a concrete type, TypeAssert checks whether the dynamic type in interface X is equal to it, and if so, the result of the conversion is a copy of the value in the interface.

If AssertedType is an interface, TypeAssert checks whether the dynamic type of the interface is assignable to it, and if so, the result of the conversion is a copy of the interface value X. If AssertedType is a superinterface of X.Type(), the operation will fail iff the operand is nil. (Contrast with ChangeInterface, which performs no nil-check.)

Type() reflects the actual type of the result, possibly a 2-types.Tuple; AssertedType is the asserted type.

Depending on the TypeAssert's purpose, Pos may return:

- the ast.CallExpr.Lparen of an explicit T(e) conversion;
- the ast.TypeAssertExpr.Lparen of an explicit e.(T) operation;
- the ast.CaseClause.Case of a case of a type-switch statement;
- the Ident(m).NamePos of an interface method value i.m (for which TypeAssert may be used to effect the nil check).

Example printed form:

```
t1 = typeassert t0.(int)
t3 = typeassert,ok t2.(T)
```

#### func (\*TypeAssert) Name ¶

```
func (v *TypeAssert) Name() string
```

#### func (\*TypeAssert) Operands ¶

```
func (v *TypeAssert) Operands(rands []*Value) []*Value
```

#### func (\*TypeAssert) Pos ¶

```
func (v *TypeAssert) Pos() token.Pos
```

#### func (\*TypeAssert) Referrers ¶

```
func (v *TypeAssert) Referrers() *[]Instruction
```

#### func (\*TypeAssert) String ¶

```
func (v *TypeAssert) String() string
```

#### func (\*TypeAssert) Type ¶

```
func (v *TypeAssert) Type() types.Type
```

#### type UnOp ¶

```
type UnOp struct {
    Op      token.Token // One of: NOT SUB ARROW MUL XOR ! - <- * ^
    X       Value
    CommaOk bool
    // contains filtered or unexported fields
}
```

The UnOp instruction yields the result of Op X. ARROW is channel receive. MUL is pointer indirection (load). XOR is bitwise complement. SUB is negation. NOT is logical negation.

If CommaOk and Op=ARROW, the result is a 2-tuple of the value above and a boolean indicating the success of the receive. The components of the tuple are accessed using Extract.

Pos() returns the ast.UnaryExpr.OpPos, if explicit in the source. For receive operations (ARROW) implicit in ranging over a channel, Pos() returns the ast.RangeStmt.For. For implicit memory loads (STAR), Pos() returns the position of the most closely associated source-level construct; the details are not specified.

Example printed form:

```
t0 = *x
t2 = <-t1,ok
```

#### func (\*UnOp) Name ¶

```
func (v *UnOp) Name() string
```

#### func (\*UnOp) Operands ¶

```
func (v *UnOp) Operands(rands []*Value) []*Value
```

#### func (\*UnOp) Pos ¶

```
func (v *UnOp) Pos() token.Pos
```

#### func (\*UnOp) Referrers ¶

```
func (v *UnOp) Referrers() *[]Instruction
```

#### func (\*UnOp) String ¶

```
func (v *UnOp) String() string
```

#### func (\*UnOp) Type ¶

```
func (v *UnOp) Type() types.Type
```

#### type Value ¶

```
type Value interface {
    // Name returns the name of this value, and determines how
    // this Value appears when used as an operand of an
    // Instruction.
    //
    // This is the same as the source name for Parameters,
    // Builtins, Functions, FreeVars, Globals.
    // For constants, it is a representation of the constant's value
    // and type.  For all other Values this is the name of the
    // virtual register defined by the instruction.
    //
    // The name of an SSA Value is not semantically significant,
    // and may not even be unique within a function.
    Name() string

    // If this value is an Instruction, String returns its
    // disassembled form; otherwise it returns unspecified
    // human-readable information about the Value, such as its
    // kind, name and type.
    String() string

    // Type returns the type of this value.  Many instructions
    // (e.g. IndexAddr) change their behaviour depending on the
    // types of their operands.
    Type() types.Type

    // Parent returns the function to which this Value belongs.
    // It returns nil for named Functions, Builtin, Const and Global.
    Parent() *Function

    // Referrers returns the list of instructions that have this
    // value as one of their operands; it may contain duplicates
    // if an instruction has a repeated operand.
    //
    // Referrers actually returns a pointer through which the
    // caller may perform mutations to the object's state.
    //
    // Referrers is currently only defined if Parent()!=nil,
    // i.e. for the function-local values FreeVar, Parameter,
    // Functions (iff anonymous) and all value-defining instructions.
    // It returns nil for named Functions, Builtin, Const and Global.
    //
    // Instruction.Operands contains the inverse of this relation.
    Referrers() *[]Instruction

    // Pos returns the location of the AST token most closely
    // associated with the operation that gave rise to this value,
    // or token.NoPos if it was not explicit in the source.
    //
    // For each ast.Node type, a particular token is designated as
    // the closest location for the expression, e.g. the Lparen
    // for an *ast.CallExpr.  This permits a compact but
    // approximate mapping from Values to source positions for use
    // in diagnostic messages, for example.
    //
    // (Do not use this position to determine which Value
    // corresponds to an ast.Expr; use Function.ValueForExpr
    // instead.  NB: it requires that the function was built with
    // debug information.)
    Pos() token.Pos
}
```

A Value is an SSA value that can be referenced by an instruction.

## Directories

| Path | Synopsis |
| --- | --- |
| [interp](https://pkg.go.dev/golang.org/x/tools@v0.48.0/go/ssa/interp)  Package ssa/interp defines an interpreter for the SSA representation of Go programs. | Package ssa/interp defines an interpreter for the SSA representation of Go programs. |
| [ssautil](https://pkg.go.dev/golang.org/x/tools@v0.48.0/go/ssa/ssautil) |  |