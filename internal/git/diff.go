package git

import (
	"fmt"
	"os/exec"
)

// GetDiff runs git diff and returns the output as a string.
// You can pass "HEAD~1" to see the last commit or "main" to compare branches.
func GetDiff(mode string) (string, error) {
	var args []string

	switch mode {
	case "staged":
		// Check if HEAD exists
		if err := exec.Command("git", "rev-parse", "HEAD").Run(); err != nil {
			// If HEAD doesn't exist (empty repo), we compare against an empty tree
			args = []string{"diff", "--cached", "4b825dc642cb6eb9a060e54bf8d69288fbee4904", "--unified=3"}
		} else {
			args = []string{"diff", "--cached", "--unified=3"}
		}
	case "last-commit":
		args = []string{"diff", "HEAD~1", "--unified=3"}
	default:
		args = []string{"diff", "--unified=3"}
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git error: %s", string(output))
	}

	return string(output), nil
}