package tools

import (
	"reflect"
	"testing"
)

func TestExtractFunctionNames(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected []string
	}{{name: "Complete Go file", code: "package main\nfunc TestOne() {}\nfunc TestTwo() {}", expected: []string{"TestOne", "TestTwo"}}, {name: "Snippet without package declaration", code: "func HelperFunc(x int) int { return x }", expected: []string{"HelperFunc"}}, {name: "Empty code string", code: "", expected: nil}, {name: "Invalid Go syntax", code: "func { invalid", expected: nil}, {name: "Mixed functions and other declarations", code: "package main\nvar x = 10\nfunc MyFunc() {}\ntype T struct{}", expected: []string{"MyFunc"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractFunctionNames(tt.code)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ExtractFunctionNames() = %v, want %v", got, tt.expected)
			}
		})
	}
}
