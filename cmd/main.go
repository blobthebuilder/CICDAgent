package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blobthebuilder/CICDAgent/internal/agent"
	"github.com/blobthebuilder/CICDAgent/internal/git"
	"github.com/blobthebuilder/CICDAgent/internal/tools"
	"github.com/joho/godotenv"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

    err := godotenv.Load()
    if err != nil {
        // slog for system-level warnings (key-value pair style)
        slog.Warn("Could not load .env file", "error", err)
    }

    // Default to "staged" mode, but allow overriding from command-line argument
    diffMode := "staged"
    if len(os.Args) > 1 {
        diffMode = os.Args[1]
    }

    // fmt for human-facing CLI headers
    fmt.Printf("🤖 CICD Agent: Analyzing changes in '%s' mode...\n", diffMode)

    diff, err := git.GetDiff(ctx, diffMode)
    if err != nil {
        slog.Error("Failed to read git diff", "error", err)
        os.Exit(1)
    }

    if diff == "" {
        fmt.Println("No changes found to analyze in the selected mode.")
        return
    }

    // 1. Get the Review and Code from the Agent
    response, err := agent.GetAction(ctx, diff)
    if err != nil {
        slog.Error("AI Generation failed", "error", err)
        os.Exit(1)
    }

    fmt.Println("\n--- 📝 AI CODE REVIEW ---")
    fmt.Println(response.Review)

	if len(response.Tests) == 0 {
		slog.Info("No tests were generated.\n")
		return
	}

	// 2. Write tests and run tests
    maxAttempts := 3
    success := false

    for attempt := 1; attempt <= maxAttempts; attempt++ {
        fmt.Printf("\n--- 🛠️ GENERATING TESTS (Attempt %d/%d) ---\n", attempt, maxAttempts)
        
        if len(response.Tests) == 0 {
            slog.Warn("Agent failed to generate a fix and returned no tests. Aborting.")
            break // Exit the loop; success remains false.
        }

        // Write the files
        testDirs := make(map[string]struct{}) // Use a map to store unique directories for targeted testing
        for _, test := range response.Tests {
            writtenPath, err := tools.WriteTestFile(test.FileName, test.Code)
            if err != nil {
                slog.Error("Failed to save test file", "file", test.FileName, "error", err)
                os.Exit(1) 
            }
            testDirs[filepath.Dir(writtenPath)] = struct{}{}
            fmt.Printf("✅ Saved test file to %s\n", writtenPath)
        }

        // Run the tests
        dirsToTest := make([]string, 0, len(testDirs))
        for dir := range testDirs {
            dirsToTest = append(dirsToTest, "./"+filepath.ToSlash(dir))
        }
        fmt.Printf("🏃 Running 'go test' on: %s ...\n", strings.Join(dirsToTest, " "))
        testResult, err := tools.RunGoTests(ctx, dirsToTest...)
        if err != nil {
            slog.Error("System error while executing tests", "error", err)
            os.Exit(1)
        }

        if testResult.Passed {
            fmt.Println("✅ ALL TESTS PASSED!")
            success = true
            break // Exit the loop!
        } 

        // IF WE REACH HERE, THE TESTS FAILED
        fmt.Println("❌ TESTS FAILED:")
        fmt.Println(testResult.Output)

        if attempt == maxAttempts {
            slog.Error("Agent failed to fix the tests after maximum attempts.", "max_attempts", maxAttempts)
            break
        }

        fmt.Println("🧠 Agent is analyzing the error and attempting a fix...")
        
        // Call the new Fix function, passing the error output back to the AI
		response, err = agent.FixTests(ctx, diff, response.Tests, testResult.Output)
		if err != nil {
            slog.Error("AI failed to generate a fix", "error", err)
            os.Exit(1)
        }

		// === NEW: THE BRANCHING LOGIC ===
        if response.CodeBugFound {
            fmt.Println("\n🚨 AGENT FOUND A BUG IN YOUR SOURCE CODE 🚨")
            fmt.Printf("Diagnosis: %s\n", response.BugExplanation)
            fmt.Println("Pipeline halted. Please fix the source code and try again.")
            success = false
            break // Exit the loop immediately
        }
    }

    if success {
        fmt.Println("\n CI Pipeline Complete: Code reviewed and tests generated successfully.")
    } else {
        fmt.Println("\n CI Pipeline Failed: Manual intervention required.")
        os.Exit(1)
    }
}