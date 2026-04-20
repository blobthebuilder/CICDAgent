package tools

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// WriteTestFile intelligently updates a Go test file by adding new imports and code.
// If the file exists, it parses it and injects the new content.
// If the file does not exist, it creates a new one with the correct package and content.
func WriteTestFile(filename string, newImports string, newCode string) (string, error) {
	// 1. Clean the path and perform security checks
	destPath := filepath.Clean(filename)
	if strings.Contains(destPath, "..") {
		return "", fmt.Errorf("security violation: path traversal '..' detected in '%s'", filename)
	}
	if filepath.IsAbs(destPath) {
		return "", fmt.Errorf("security violation: absolute paths are not allowed: '%s'", filename)
	}
	if !strings.HasSuffix(destPath, "_test.go") {
		return "", fmt.Errorf("security violation: filename '%s' must end in _test.go", destPath)
	}

	fset := token.NewFileSet()
	var node *ast.File
	var err error

	// 2. Parse existing file or create a new AST if it doesn't exist
	if _, statErr := os.Stat(destPath); os.IsNotExist(statErr) {
		// File doesn't exist. We need to create a new AST.
		// We need the package name from the corresponding source file.
		srcFile := strings.TrimSuffix(destPath, "_test.go") + ".go"
		// Parse only the package clause to be efficient.
		srcNode, parseErr := parser.ParseFile(fset, srcFile, nil, parser.PackageClauseOnly)
		if parseErr != nil {
			return "", fmt.Errorf("failed to parse source file '%s' to determine package name: %w", srcFile, parseErr)
		}
		if srcNode.Name == nil {
			return "", fmt.Errorf("could not determine package name from '%s'", srcFile)
		}

		node = &ast.File{
			Name: srcNode.Name,
		}
	} else {
		// File exists, parse it fully.
		node, err = parser.ParseFile(fset, destPath, nil, parser.ParseComments)
		if err != nil {
			return "", fmt.Errorf("failed to parse existing test file '%s': %w", destPath, err)
		}
	}

	// 3. Add new imports
	importsToAdd := strings.Split(newImports, "\n")
	for _, impPath := range importsToAdd {
		unquotedPath := strings.TrimSpace(strings.Trim(impPath, `"`))
		if unquotedPath != "" {
			astutil.AddImport(fset, node, unquotedPath)
		}
	}

	// 4. Add or update functions/declarations from newCode
	if strings.TrimSpace(newCode) != "" {
		wrappedCode := fmt.Sprintf("package %s\n\n%s", node.Name.Name, newCode)

		tempFset := token.NewFileSet()
		tempNode, err := parser.ParseFile(tempFset, "", wrappedCode, parser.ParseComments)
		if err != nil {
			return "", fmt.Errorf("failed to parse new code block: %w\n---Code Block---\n%s", err, newCode)
		}

		// Collect names of functions in the new code
		newFuncNames := make(map[string]bool)
		for _, newDecl := range tempNode.Decls {
			if newFn, isFn := newDecl.(*ast.FuncDecl); isFn {
				newFuncNames[newFn.Name.Name] = true
			}
		}

		// Filter out existing functions with those names and capture their position spans
		var filteredDecls []ast.Decl
		type span struct{ start, end token.Pos }
		var removedSpans []span

		for _, existingDecl := range node.Decls {
			if existingFn, isFn := existingDecl.(*ast.FuncDecl); isFn && newFuncNames[existingFn.Name.Name] {
				start := existingFn.Pos()
				if existingFn.Doc != nil {
					start = existingFn.Doc.Pos()
				}
				removedSpans = append(removedSpans, span{start, existingFn.End()})
				continue
			}
			filteredDecls = append(filteredDecls, existingDecl)
		}
		node.Decls = filteredDecls

		// Remove comments that belonged to the replaced functions
		var filteredComments []*ast.CommentGroup
		for _, cg := range node.Comments {
			keep := true
			for _, s := range removedSpans {
				if cg.Pos() >= s.start && cg.End() <= s.end {
					keep = false
					break
				}
			}
			if keep {
				filteredComments = append(filteredComments, cg)
			}
		}
		node.Comments = filteredComments
	}

	// 5. Format the modified AST back to a buffer
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return "", fmt.Errorf("failed to format updated AST for '%s': %w", destPath, err)
	}

	// 6. Append the new code text and format the entire file
	if strings.TrimSpace(newCode) != "" {
		buf.WriteString("\n\n")
		buf.WriteString(newCode)
	}

	finalSource, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to format final source code: %w\n%s", err, buf.String())
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory for '%s': %w", destPath, err)
	}
	if err := os.WriteFile(destPath, finalSource, 0644); err != nil {
		return "", fmt.Errorf("failed to write updated file '%s': %w", destPath, err)
	}

	return destPath, nil
}

func ReadFile(filename string) (string, error) {
	// 1. Clean the path and perform security checks to prevent reading files outside the project.
	cleanPath := filepath.Clean(filename)
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("security violation: path traversal '..' detected in '%s'", filename)
	}
	if filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("security violation: absolute paths are not allowed: '%s'", filename)
	}

	// 2. Don't let the AI read your .env!
	if filepath.Base(cleanPath) == ".env" {
		return "", fmt.Errorf("access denied")
	}
	content, err := os.ReadFile(cleanPath)
	return string(content), err
}
