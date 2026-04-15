package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteTestFile(filename string, content string) (string, error) {
	// 1. Clean the path and perform security checks
	destPath := filepath.Clean(filename)
	if strings.Contains(destPath, "..") {
		return "", fmt.Errorf("security violation: path traversal '..' detected in '%s'", filename)
	}
	if filepath.IsAbs(destPath) {
		return "", fmt.Errorf("security violation: absolute paths are not allowed: '%s'", filename)
	}

	// 2. Enforce the test suffix on the final path component
	if !strings.HasSuffix(destPath, "_test.go") {
		return "", fmt.Errorf("security violation: filename '%s' must end in _test.go", destPath)
	}

	// 3. Ensure the target directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 4. Write the file. This will create a new file or overwrite an existing one.
	err := os.WriteFile(destPath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
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
