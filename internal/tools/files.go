package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteTestFile(filename string, content string) error {
	// 1. Clean the path to prevent directory traversal (e.g., "../../etc/passwd")
	safeName := filepath.Base(filename)
	
	// 2. Enforce the test suffix
	if !strings.HasSuffix(safeName, "_test.go") {
		return fmt.Errorf("security violation: filename '%s' must end in _test.go", safeName)
	}

	// 3. Prevent overwriting existing files (optional but recommended)
	if _, err := os.Stat(safeName); err == nil {
		return fmt.Errorf("security violation: file '%s' already exists; refusing to overwrite", safeName)
	}

	// 4. Final write (using 0644 - read/write for user, read-only for others)
	err := os.WriteFile(safeName, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func ReadFile(filename string) (string, error) {
    // Basic safety: don't let the AI read your .env!
    if filename == ".env" {
        return "", fmt.Errorf("access denied")
    }
    content, err := os.ReadFile(filename)
    return string(content), err
}