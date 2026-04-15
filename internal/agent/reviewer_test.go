package agent

import (
	"strings"
	"testing"
)

func TestParseAgentResponse(t *testing.T) {
	t.Run("Standard well-formatted response", func(t *testing.T) {
		rawJSON := `{
			"review": "The code looks good.",
			"tests": [
				{
					"file_name": "internal/logic/math_test.go",
					"code": "package logic\nfunc TestAdd(t *testing.T) {}"
				}
			]
		}`

		resp, err := parseAgentResponse(rawJSON)
		if err != nil {
			t.Fatalf("parseAgentResponse failed: %v", err)
		}

		if resp.Review != "The code looks good." {
			t.Errorf("Expected review 'The code looks good.', got '%s'", resp.Review)
		}
		if len(resp.Tests) != 1 {
			t.Fatalf("Expected 1 test, got %d", len(resp.Tests))
		}
		if resp.Tests[0].FileName != "internal/logic/math_test.go" {
			t.Errorf("Expected filename 'internal/logic/math_test.go', got '%s'", resp.Tests[0].FileName)
		}
		if resp.Tests[0].Code != "package logic\nfunc TestAdd(t *testing.T) {}" {
			t.Errorf("Expected code 'package logic...', got '%s'", resp.Tests[0].Code)
		}
	})

	t.Run("Response with multiple files", func(t *testing.T) {
		rawJSON := `{
			"review": "Review for two files.",
			"tests": [
				{ "file_name": "file1_test.go", "code": "package one" },
				{ "file_name": "file2_test.go", "code": "package two" }
			]
		}`

		resp, err := parseAgentResponse(rawJSON)
		if err != nil {
			t.Fatalf("parseAgentResponse failed: %v", err)
		}

		if len(resp.Tests) != 2 {
			t.Fatalf("Expected 2 tests, got %d", len(resp.Tests))
		}
		if resp.Tests[0].FileName != "file1_test.go" || resp.Tests[0].Code != "package one" {
			t.Errorf("Mismatch in first test file data")
		}
		if resp.Tests[1].FileName != "file2_test.go" || resp.Tests[1].Code != "package two" {
			t.Errorf("Mismatch in second test file data")
		}
	})

	t.Run("Response with no tests", func(t *testing.T) {
		rawJSON := `{"review": "No tests needed.", "tests": []}`
		resp, err := parseAgentResponse(rawJSON)
		if err != nil {
			t.Fatalf("parseAgentResponse failed: %v", err)
		}
		if resp.Review != "No tests needed." {
			t.Errorf("Expected review 'No tests needed.', got '%s'", resp.Review)
		}
		if len(resp.Tests) != 0 {
			t.Errorf("Expected 0 tests, got %d", len(resp.Tests))
		}
	})

	t.Run("Response wrapped in markdown", func(t *testing.T) {
		rawJSON := "```json\n" + `{"review": "wrapped", "tests": []}` + "\n```"
		resp, err := parseAgentResponse(rawJSON)
		if err != nil {
			t.Fatalf("parseAgentResponse failed for markdown-wrapped JSON: %v", err)
		}
		if resp.Review != "wrapped" {
			t.Errorf("Expected review 'wrapped', got '%s'", resp.Review)
		}
	})

	t.Run("Malformed JSON response", func(t *testing.T) {
		rawJSON := `{"review": "malformed", "tests": [`
		_, err := parseAgentResponse(rawJSON)
		if err == nil {
			t.Error("Expected an error for malformed JSON, but got nil")
		}
		if !strings.Contains(err.Error(), "failed to unmarshal JSON") {
			t.Errorf("Expected unmarshal error, got: %v", err)
		}
	})
}