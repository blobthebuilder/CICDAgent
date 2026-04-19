package tools

import (
	"os"
	"testing"
	"strings"
)

func TestExtractGoFileInfo(t *testing.T) {
	content := "package testpkg\nimport \"fmt\"\nfunc Hello() { fmt.Println(\"hi\") }\ntype T struct{}\n"
	tmpfile, err := os.CreateTemp("", "ast_test_*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	info, err := ExtractGoFileInfo(tmpfile.Name())
	if err != nil {
		t.Fatalf("ExtractGoFileInfo failed: %v", err)
	}
	if !strings.Contains(info, "package testpkg") {
		t.Errorf("expected package name in output")
	}
	if !strings.Contains(info, "func Hello()") {
		t.Errorf("expected function signature in output")
	}
	if strings.Contains(info, "fmt.Println") {
		t.Errorf("expected function body to be stripped")
	}
}
