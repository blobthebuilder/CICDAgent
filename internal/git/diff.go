package git

import (
	"context"
	"fmt"
	"os/exec"
)

// GetDiff runs git diff and returns the output as a string.
// You can pass "HEAD~1" to see the last commit or "main" to compare branches.
func GetDiff(ctx context.Context, mode string) (string, error) {
	var args []string

	switch mode {
	case "staged":
		// Check if HEAD exists
		if err := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Run(); err != nil {
			// If HEAD doesn't exist (empty repo), we compare against an empty tree
			args = []string{"diff", "--cached", "4b825dc642cb6eb9a060e54bf8d69288fbee4904", "--unified=3"}
		} else {
			args = []string{"diff", "--cached", "--unified=3"}
		}
	case "last-commit": // this compares only changes in last commit
		// Check if HEAD~1 exists to avoid crashing on the very first commit
		if err := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "HEAD~1").Run(); err != nil {
			// If HEAD~1 doesn't exist, we're on the initial commit. Diff against the empty tree.
			args = []string{"diff", "4b825dc642cb6eb9a060e54bf8d69288fbee4904", "HEAD", "--unified=3"}
		} else {
			args = []string{"diff", "HEAD~1", "HEAD", "--unified=3"}
		}
	case "full": // this compares the entire codebase from the start
		// 4b825dc642cb6eb9a060e54bf8d69288fbee4904 is the hash of an empty git tree
		if err := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Run(); err != nil {
			// No commits yet, diff staged files against empty tree
			args = []string{"diff", "--cached", "4b825dc642cb6eb9a060e54bf8d69288fbee4904", "--unified=3"}
		} else {
			args = []string{"diff", "4b825dc642cb6eb9a060e54bf8d69288fbee4904", "HEAD", "--unified=3"}
		}
	default:
		args = []string{"diff", "--unified=3"}
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git error: %s", string(output))
	}

	return string(output), nil
}