package agent

import (
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