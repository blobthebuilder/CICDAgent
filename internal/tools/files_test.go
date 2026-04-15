package tools

import (
	"os"
	"testing"
)

func TestWriteTestFile_Security(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{"Valid test file", "example_test.go", false},
		{"Path traversal", "../../etc/passwd_test.go", true},
		{"Absolute path", "/tmp/test_test.go", true},
		{"Missing suffix", "example.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := WriteTestFile(tt.filename, "package tools")
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteTestFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				os.Remove(path)
			}
		})
	}
}

func TestReadFile_Security(t *testing.T) {
	// Create a dummy .env file for testing
	err := os.WriteFile(".env", []byte("SECRET=123"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(".env")

	t.Run("Access denied to .env", func(t *testing.T) {
		_, err := ReadFile(".env")
		if err == nil || err.Error() != "access denied" {
			t.Errorf("Expected access denied error for .env, got %v", err)
		}
	})

	t.Run("Path traversal check", func(t *testing.T) {
		_, err := ReadFile("../reviewer.go")
		if err == nil {
			t.Error("Expected error for path traversal")
		}
	})
}