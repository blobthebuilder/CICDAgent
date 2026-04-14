package tools

import (
	"os"
)

// WriteTestFile creates a new test file with the provided content.
func WriteTestFile(filename string, content string) error {
	// Ensure we don't overwrite important source code by accident
	// You might want to enforce a _test.go suffix later
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return err
	}
	return nil
}