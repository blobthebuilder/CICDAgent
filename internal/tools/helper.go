package tools

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// extractFunctionNames parses Go code and returns a list of function names defined in it.
func ExtractFunctionNames(code string) []string {
	var names []string
	fset := token.NewFileSet()
	
	// First, attempt to parse the code as a complete file
	node, err := parser.ParseFile(fset, "", code, 0)
	if err != nil {
		// If parsing fails, it might be a snippet missing a package declaration.
		// Wrap the snippet in a dummy package so it forms a valid Go file for the parser.
		src := "package dummy\n\n" + code
		node, err = parser.ParseFile(fset, "", src, 0)
		if err != nil {
			return names // Return what we can if parsing still fails
		}
	}

	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			names = append(names, fn.Name.Name)
		}
	}
	return names
}