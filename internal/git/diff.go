package git

import (
	"os/exec"
)

// GetDiff runs git diff and returns the output as a string.
// You can pass "HEAD~1" to see the last commit or "main" to compare branches.
func GetDiff(target string) (string, error) {
	// We use --unified=3 to give the AI some surrounding context lines
	cmd := exec.Command("git", "diff", target, "--unified=3")
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	
	return string(output), nil
}