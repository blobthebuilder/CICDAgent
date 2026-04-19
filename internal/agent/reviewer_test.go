package agent

import (
	"os"
	"strings"
	"testing"
)

func TestParseAgentResponse(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expectOk bool
	}{
		{
			name:     "Valid JSON",
			input:    `{"review": "looks good", "tests": [{"file_name": "main_test.go", "code": "package main"}]}`,
			expectOk: true,
		},
		{
			name:     "Markdown Wrapped JSON",
			input:    "```json\n{\"review\": \"wrapped\", \"tests\": []}\n```",
			expectOk: true,
		},
		{
			name:     "Invalid JSON",
			input:    `{invalid: json}`,
			expectOk: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := parseAgentResponse(tc.input)
			if tc.expectOk && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if !tc.expectOk && err == nil {
				t.Error("expected error, got nil")
			}
			if tc.expectOk && resp == nil {
				t.Error("expected response, got nil")
			}
		})
	}
}
func TestParseFilenamesFromDiff(t *testing.T) {
	diff := "--- a/internal/agent/reviewer.go\n+++ b/internal/agent/reviewer.go\n@@ -1,1 +1,2 @@\n--- a/internal/tools/files_test.go\n+++ b/internal/tools/files_test.go\n--- a/cmd/main.go\n+++ b/cmd/main.go\n"

	files := parseFilenamesFromDiff(diff)
	expected :=
		map[string]bool{"internal/agent/reviewer.go": true, "cmd/main.go": true}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d",

			len(files))
	}
	for _, f := range files {
		if !expected[f] {
			t.
				Errorf("unexpected file found: %s",

					f)
		}
	}
}
func TestGetTestFileContext(t *testing.T) {
	tmpDir := t.TempDir()
	currDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err !=
		nil {
		t.Fatal(err)
	}
	defer os.Chdir(currDir)
	srcFile := "processor.go"
	testFile := "processor_test.go"
	srcContent := "package processor\nfunc Process() {}\n"
	testContent :=
		"package processor\nimport \"testing\"\nfunc TestProcess(t *testing.T) {}\n"

	if err := os.WriteFile(srcFile, []byte(srcContent),
		0644); err !=
		nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFile,
		[]byte(testContent), 0644,
	); err != nil {
		t.Fatal(err)
	}
	diff := "--- a/processor.go\n+++ b/processor.go\n"
	ctxResult := getTestFileContext(diff)
	if !strings.Contains(ctxResult,
		"--- Existing Test File: processor_test.go ---",
	) {
		t.Errorf("expected context to contain test file header, got: %s",

			ctxResult,
		)
	}
	if !strings.Contains(ctxResult,
		"func TestProcess(t *testing.T)") {
		t.Errorf("expected context to contain function signature, got: %s",

			ctxResult)
	}
}
