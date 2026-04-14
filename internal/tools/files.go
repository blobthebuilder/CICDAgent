package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteTestFile(filename string, content string) (string, error) {
	// Get user-defined directory from environment, default to current directory.
	outputDir := os.Getenv("TEST_OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "."
	}

	// 1. Clean the path to prevent directory traversal (e.g., "../../etc/passwd")
	safeName := filepath.Base(filename)

	// 2. Enforce the test suffix
	if !strings.HasSuffix(safeName, "_test.go") {
		return "", fmt.Errorf("security violation: filename '%s' must end in _test.go", safeName)
	}

	// 3. Construct the full destination path
	destPath := filepath.Join(outputDir, safeName)

	// 4. Ensure the target directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 5. Prevent overwriting existing files (optional but recommended)
	if _, err := os.Stat(destPath); err == nil {
		return "", fmt.Errorf("security violation: file '%s' already exists; refusing to overwrite", destPath)
	}

	// 6. Final write (using 0644 - read/write for user, read-only for others)
	err := os.WriteFile(destPath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return destPath, nil
}

func ReadFile(filename string) (string, error) {
    // Basic safety: don't let the AI read your .env!
    if filepath.Base(filename)== ".env" {
        return "", fmt.Errorf("access denied")
    }
    content, err := os.ReadFile(filename)
    return string(content), err
}
