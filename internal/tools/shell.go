package tools

import (
	"bytes"
	"os/exec"
)

// TestResult holds the output of the go test command
type TestResult struct {
	Passed bool
	Output string // The terminal output (errors or success messages)
}

// RunGoTests executes 'go test' on a specific set of directories and captures the result.
// If no paths are provided, it defaults to running all tests with './...'.
func RunGoTests(paths ...string) (*TestResult, error) {
	args := []string{"test"}
	if len(paths) > 0 {
		args = append(args, paths...)
	}
	// Create the command
	cmd := exec.Command("go", args...)

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