package tools

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunGoTests_NonExistentPackage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Running tests on a directory that clearly doesn't exist should fail compilation/execution
	result, err := RunGoTests(ctx, "./non_existent_directory_12345")
	if err != nil {
		t.Fatalf("RunGoTests should return nil error on test failure, got %v", err)
	}

	if result.Passed {
		t.Error("Expected Passed to be false for a non-existent package")
	}

	if result.Output == "" {
		t.Error("Expected output to contain error messages, but it was empty")
	}
}
func TestRunGoStaticAnalysis_InvalidPath(
	t *testing.T) {
	ctx, cancel := context.WithTimeout(context.
		Background(), 2*time.Second)
	defer cancel()

	output, err := RunGoStaticAnalysis(ctx, "./non_existent_path_test")
	if err == nil {
		t.Fatal("expected error for non-existent path, got nil")
	}
	if !strings.Contains(output, "go fmt") {
		t.Errorf("expected output to mention 'go fmt', got: %s", output)

	}
}
func TestRunGoStaticAnalysis_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(
		context.Background())
	cancel()
	_, err := RunGoStaticAnalysis(ctx, ".")
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
}
