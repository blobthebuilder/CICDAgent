package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// TestResult holds the output of the go test command
type TestResult struct {
	Passed bool
	Output string // The terminal output (errors or success messages)
}

// RunGoTests executes 'go test' on a specific set of directories and captures the result.
// If no paths are provided, it defaults to running all tests with './...'.
func RunGoTests(ctx context.Context, paths ...string) (*TestResult, error) {
	args := []string{"test"}
	if len(paths) > 0 {
		args = append(args, paths...)
	}
	// Create the command
	cmd := exec.CommandContext(ctx, "go", args...)

	// We need to capture the output, not just let it print to the terminal
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	// Run the command
	err := cmd.Run()

	result := &TestResult{
		Output: out.String(),
	}

	if err != nil {
		// If err is not nil, it usually means the tests failed or didn't compile
		result.Passed = false
		return result, nil // We return nil for the error because test failure isn't an app crash
	}

	result.Passed = true
	return result, nil
}

// RunGoStaticAnalysis executes 'go fmt' and 'go vet' on a specific set of directories.
// It returns an error if any of the static analysis tools report issues.
func RunGoStaticAnalysis(ctx context.Context, paths ...string) (string, error) {
	var outputBuilder strings.Builder

	// 1. Run go fmt
	fmtArgs := []string{"fmt"}
	if len(paths) > 0 {
		fmtArgs = append(fmtArgs, paths...)
	} else {
		fmtArgs = append(fmtArgs, "./...") // Default to current module
	}
	fmtCmd := exec.CommandContext(ctx, "go", fmtArgs...)
	fmtOut, err := fmtCmd.CombinedOutput()
	if err != nil {
		outputBuilder.WriteString(fmt.Sprintf("`go fmt` failed:\n%s\nError: %v\n", string(fmtOut), err))
		return outputBuilder.String(), fmt.Errorf("`go fmt` failed: %w", err)
	}
	if len(fmtOut) > 0 {
		outputBuilder.WriteString(fmt.Sprintf("`go fmt` output:\n%s\n", string(fmtOut)))
	}

	// 2. Run go vet
	vetArgs := []string{"vet"}
	if len(paths) > 0 {
		vetArgs = append(vetArgs, paths...)
	} else {
		vetArgs = append(vetArgs, "./...") // Default to current module
	}
	vetCmd := exec.CommandContext(ctx, "go", vetArgs...)
	vetOut, err := vetCmd.CombinedOutput()
	if err != nil {
		outputBuilder.WriteString(fmt.Sprintf("`go vet` failed:\n%s\nError: %v\n", string(vetOut), err))
		return outputBuilder.String(), fmt.Errorf("`go vet` reported issues: %w", err)
	}
	outputBuilder.WriteString(fmt.Sprintf("`go vet` output:\n%s\n", string(vetOut)))
	return outputBuilder.String(), nil
}
