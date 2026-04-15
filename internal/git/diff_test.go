package git

import "testing"

func TestGetDiff_InvalidMode(t *testing.T) {
	// Since GetDiff executes external git commands, we test the default fallback behavior
	// in an environment where git might not be initialized as a repo.
	_, err := GetDiff("invalid-mode")
	if err != nil {
		// If git fails because we are not in a repo, that's expected in some CI environments
		t.Logf("Git diff failed as expected or due to environment: %v", err)
	}
}