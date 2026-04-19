package tools

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
)

// ExtractGoFileInfo parses a Go source file and extracts package, imports,
// declarations, and function signatures into a string for LLM context.
// It reads the file from the filesystem.
func ExtractGoFileInfo(filename string) (string, error) {
	fset := token.NewFileSet()
	// Pass nil for src, so parser will read the file from disk.
	node, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return "", fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	var out bytes.Buffer

	// 1. Package name
	out.WriteString(fmt.Sprintf("package %s\n\n", node.Name.Name))

	// 2. Declarations (imports, consts, vars, types, funcs)
	for _, decl := range node.Decls {
		var buf bytes.Buffer

		// For functions, nil out the body to get only the signature.
		if fn, ok := decl.(*ast.FuncDecl); ok {
			fn.Body = nil
		}

		// Format the node (either full GenDecl or FuncDecl with no body)
		if err := format.Node(&buf, fset, decl); err != nil {
			return "", fmt.Errorf("failed to format AST node: %w", err)
		}

		out.Write(buf.Bytes())
		out.WriteString("\n\n")
	}

	return out.String(), nil
}
