package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteTestFile(t *testing.T) {
	// Setup temporary directory for testing
	tmpDir := t.TempDir()
	t.Setenv("TEST_OUTPUT_DIR", tmpDir)

	t.Run("Successfully write a valid test file", func(t *testing.T) {
		filename := "logic_test.go"
		content := "package main\n\nfunc TestLogic(t *testing.T) {}"

		path, err := WriteTestFile(filename, content)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify path
		expectedPath := filepath.Join(tmpDir, "logic_test.go")
		if path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, path)
		}

		// Verify content
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read created file: %v", err)
		}
		if string(data) != content {
			t.Errorf("Expected content %s, got %s", content, string(data))
		}
	})

	t.Run("Reject file without _test.go suffix", func(t *testing.T) {
		filename := "malicious.go"
		content := "package main"

		_, err := WriteTestFile(filename, content)
		if err == nil {
			t.Fatal("Expected error for non-test suffix, got nil")
		}
		if err.Error() != "security violation: filename 'malicious.go' must end in _test.go" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("Prevent overwriting existing file", func(t *testing.T) {
		filename := "duplicate_test.go"
		content := "package main"

		// Write once
		_, err := WriteTestFile(filename, content)
		if err != nil {
			t.Fatalf("First write failed: %v", err)
		}

		// Try to write again
		_, err = WriteTestFile(filename, content)
		if err == nil {
			t.Fatal("Expected error when overwriting, got nil")
		}
	})

	t.Run("Directory traversal protection via Base", func(t *testing.T) {
		// Even if AI tries to escape, filepath.Base should strip it
		filename := "../../../etc/passwd_test.go"
		content := "package main"

		path, err := WriteTestFile(filename, content)
		if err != nil {
			t.Fatalf("Expected no error due to filepath.Base stripping, got %v", err)
		}

		if filepath.Base(path) != "passwd_test.go" {
			t.Errorf("Security check failed: filename was not properly sanitized. Got: %s", path)
		}
	})
}

func TestReadFile(t *testing.T) {
	t.Run("Prevent reading .env", func(t *testing.T) {
		_, err := ReadFile(".env")
		if err == nil {
			t.Fatal("Expected access denied for .env, got nil")
		}
		if err.Error() != "access denied" {
			t.Errorf("Expected 'access denied', got: %v", err)
		}
	})

	t.Run("Read valid file", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "hello.txt")
		content := "hello world"
		err := os.WriteFile(tmpFile, []byte(content), 0644)
		if err != nil {
			t.Fatal(err)
		}

		got, err := ReadFile(tmpFile)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if got != content {
			t.Errorf("Expected %s, got %s", content, got)
		}
	})
}