package tools

import (
	"testing"
	"os"
	"path/filepath"
)

func TestRunGoTests(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "testrepo")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a dummy go file to test against
	goMod := "module testrepo\n\ngo 1.21"
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	if err != nil {
		t.Fatal(err)
	}

	passingTest := "package testrepo\nimport \"testing\"\nfunc TestPass(t *testing.T) {}"
	err = os.WriteFile(filepath.Join(tmpDir, "pass_test.go"), []byte(passingTest), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Successful Test", func(t *testing.T) {
		origDir, _ := os.Getwd()
		err := os.Chdir(tmpDir)
		if err != nil {
			t.Fatalf("failed to change directory: %v", err)
		}
		defer os.Chdir(origDir)

		result, err := RunGoTests(".")
		if err != nil {
			t.Errorf("RunGoTests failed: %v", err)
		}
		if !result.Passed {
			t.Errorf("Expected tests to pass, got output: %s", result.Output)
		}
	})

	t.Run("Failing Test", func(t *testing.T) {
		failingTest := "package testrepo\nimport \"testing\"\nfunc TestFail(t *testing.T) { t.Fatal(\"fail\") }"
		err := os.WriteFile(filepath.Join(tmpDir, "fail_test.go"), []byte(failingTest), 0644)
		if err != nil {
			t.Fatal(err)
		}
		
		origDir, _ := os.Getwd()
		err = os.Chdir(tmpDir)
		if err != nil {
			t.Fatalf("failed to change directory: %v", err)
		}
		defer os.Chdir(origDir)

		result, err := RunGoTests(".")
		if err != nil {
			t.Errorf("RunGoTests failed: %v", err)
		}
		if result.Passed {
			t.Error("Expected tests to fail, but they passed")
		}
	})
}