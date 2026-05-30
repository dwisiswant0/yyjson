package unsafeptrpolicy

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

const (
	bindPackage      = "go.dw1.io/yyjson/internal/bind"
	generatedPackage = "go.dw1.io/yyjson/internal/yyjsonc"
)

var Analyzer = &analysis.Analyzer{
	Name: "yyjsonunsafeptr",
	Doc:  "checks yyjson's handwritten uintptr to unsafe.Pointer boundary",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if IsGeneratedPackage(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		parents := nodeParents(file)
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !isUintptrUnsafePointerConversion(pass, call) {
				return true
			}
			if isAllowedBindCPointer(pass, enclosingFunc(parents, call), call) {
				return true
			}
			pass.Reportf(call.Pos(), "uintptr to unsafe.Pointer conversion is only allowed in internal/bind.cPointer or generated yyjsonc")
			return true
		})
	}

	return nil, nil
}

// IsGeneratedPackage reports whether path is yyjson's ccgo-generated package.
func IsGeneratedPackage(path string) bool {
	return path == generatedPackage
}

func nodeParents(root ast.Node) map[ast.Node]ast.Node {
	parents := make(map[ast.Node]ast.Node)
	var stack []ast.Node
	ast.Inspect(root, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if len(stack) > 0 {
			parents[n] = stack[len(stack)-1]
		}
		stack = append(stack, n)
		return true
	})
	return parents
}

func enclosingFunc(parents map[ast.Node]ast.Node, n ast.Node) *ast.FuncDecl {
	for parent := parents[n]; parent != nil; parent = parents[parent] {
		if fn, ok := parent.(*ast.FuncDecl); ok {
			return fn
		}
	}
	return nil
}

func isUintptrUnsafePointerConversion(pass *analysis.Pass, call *ast.CallExpr) bool {
	if len(call.Args) != 1 {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Pointer" {
		return false
	}
	if !isUnsafePackage(pass, sel.X) {
		return false
	}
	return isUintptrType(pass.TypesInfo.TypeOf(call.Args[0]))
}

func isUnsafePackage(pass *analysis.Pass, expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	pkg, ok := pass.TypesInfo.Uses[id].(*types.PkgName)
	return ok && pkg.Imported() != nil && pkg.Imported().Path() == "unsafe"
}

func isUintptrType(t types.Type) bool {
	if t == nil {
		return false
	}
	return types.Identical(t, types.Typ[types.Uintptr]) ||
		types.Identical(t.Underlying(), types.Typ[types.Uintptr])
}

func isAllowedBindCPointer(pass *analysis.Pass, fn *ast.FuncDecl, call *ast.CallExpr) bool {
	if pass.Pkg.Path() != bindPackage || fn == nil || fn.Name.Name != "cPointer" {
		return false
	}
	if !isCPointerSignature(pass, fn) || !isOnlyReturnValue(fn, call) {
		return false
	}
	arg, ok := call.Args[0].(*ast.Ident)
	return ok && arg.Name == "addr"
}

func isCPointerSignature(pass *analysis.Pass, fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 ||
		len(fn.Type.Params.List[0].Names) != 1 ||
		fn.Type.Params.List[0].Names[0].Name != "addr" {
		return false
	}
	param := pass.TypesInfo.Defs[fn.Type.Params.List[0].Names[0]]
	if param == nil || !isUintptrType(param.Type()) {
		return false
	}
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	return types.Identical(pass.TypesInfo.TypeOf(fn.Type.Results.List[0].Type), types.Typ[types.UnsafePointer])
}

func isOnlyReturnValue(fn *ast.FuncDecl, call *ast.CallExpr) bool {
	if fn.Body == nil || len(fn.Body.List) != 1 {
		return false
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	return ok && len(ret.Results) == 1 && ret.Results[0] == call
}
