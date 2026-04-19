package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go/parser"
	"go/token"
)

func TestWriteTestFile_Security(t *testing.T) {
	// Create a dummy source file for the valid test case to find a package name
	if err := os.WriteFile("example.go", []byte("package tools"), 0644); err != nil {
		t.Fatalf("failed to create dummy source file: %v", err)
	}
	defer os.Remove("example.go")

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
			// The content doesn't matter for security checks, but should be valid for the non-error case.
			path, err := WriteTestFile(tt.filename, `"testing"`, "func TestExample(t *testing.T) {}")
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteTestFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				os.Remove(path)
			}
		})
	}
}

func TestWriteTestFile_AppendContent(t *testing.T) {
	// Setup: Create a directory and initial files
	// We use a relative local directory because WriteTestFile now strictly blocks absolute paths
	testDir := "temp_append_test_dir"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create relative temp dir: %v", err)
	}
	defer os.RemoveAll(testDir) // Clean up afterwards

	srcFileName := filepath.Join(testDir, "append.go")
	testFileName := filepath.Join(testDir, "append_test.go")

	initialSrcContent := "package append"
	initialTestContent := `package append

import "testing"

func TestInitial(t *testing.T) {
	// initial test
}
`
	if err := os.WriteFile(srcFileName, []byte(initialSrcContent), 0644); err != nil {
		t.Fatalf("Failed to write initial source file: %v", err)
	}
	if err := os.WriteFile(testFileName, []byte(initialTestContent), 0644); err != nil {
		t.Fatalf("Failed to write initial test file: %v", err)
	}

	// Action: Call WriteTestFile to append new content
	newImports := `"fmt"`
	newCode := `func TestAppended(t *testing.T) {
	fmt.Println("appended")
}`
	_, err := WriteTestFile(testFileName, newImports, newCode)
	if err != nil {
		t.Fatalf("WriteTestFile failed: %v", err)
	}

	// Verification: Read the file and check its content
	finalContent, err := os.ReadFile(testFileName)
	if err != nil {
		t.Fatalf("Failed to read final test file: %v", err)
	}

	finalContentStr := string(finalContent)

	// Check that both old and new content exist
	if !strings.Contains(finalContentStr, "TestInitial") {
		t.Error("Final content is missing the initial test function.")
	}
	if !strings.Contains(finalContentStr, "TestAppended") {
		t.Error("Final content is missing the appended test function.")
	}
	// Check that the new import was added.
	if !strings.Contains(finalContentStr, `"fmt"`) {
		t.Error("Final content is missing the new import.")
	}

	// Check that it's still a valid Go file
	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, "", finalContent, 0)
	if err != nil {
		t.Errorf("Final content is not a valid Go file: %v", err)
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