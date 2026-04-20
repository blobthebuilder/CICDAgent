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
        slog.Warn("Could not load .env file", "error", err)
    }

    // Default to "staged" mode, but allow overriding from command-line argument
    diffMode := "staged"
    if len(os.Args) > 1 {
        diffMode = os.Args[1]
    }

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
        var writeError error
        var failureOutput string // Declared here
        var dirsToTest []string  // Declared here
        testDirs := make(map[string]struct{}) // Use a map to store unique directories for targeted testing
        for _, test := range response.Tests {
            writtenPath, err := tools.WriteTestFile(test.FileName, test.Imports, test.Code)
            if err != nil {
                slog.Error("Failed to save test file", "file", test.FileName, "error", err)
                writeError = fmt.Errorf("failed to save test file %s: %w", test.FileName, err)
                break
            }
            testDirs[filepath.Dir(writtenPath)] = struct{}{}
            fmt.Printf("✅ Saved test file to %s\n", writtenPath)

            // Parse the generated code and list the newly added functions
            funcNames := tools.ExtractFunctionNames(test.Code)
            if len(funcNames) > 0 {
                fmt.Println("   Added/Updated functions:")
                for _, name := range funcNames {
                    fmt.Printf("   - %s\n", name)
                }
            }
        }

        if writeError != nil {
            failureOutput = writeError.Error()
        } else {
            // Populate dirsToTest here, after files are written and testDirs is populated
            dirsToTest = make([]string, 0, len(testDirs))
            for dir := range testDirs {
                dirsToTest = append(dirsToTest, "./"+filepath.ToSlash(dir))
            }

            // Run static analysis (go fmt, go vet) on the generated files
            fmt.Printf("🔍 Running static analysis on generated files in: %s ...\n", strings.Join(dirsToTest, " "))
            staticAnalysisOutput, err := tools.RunGoStaticAnalysis(ctx, dirsToTest...)
            if err != nil {
                slog.Error("Static analysis failed", "error", err)
                failureOutput = fmt.Sprintf("Static analysis failed:\n%s\n%v", staticAnalysisOutput, err)
            } else {
                if len(staticAnalysisOutput) > 0 {
                    fmt.Printf("Static analysis output:\n%s\n", staticAnalysisOutput)
                }

            // Run the tests
                fmt.Printf("🏃 Running 'go test' on: %s ...\n", strings.Join(dirsToTest, " "))
                testResult, err := tools.RunGoTests(ctx, dirsToTest...)
                if err != nil {
                    slog.Error("System error while executing tests", "error", err)
                    os.Exit(1) // This is a system error, not a test failure, so exit.
                }

                if testResult.Passed {
                    fmt.Println("✅ ALL TESTS PASSED!")
                    success = true
                    break // Exit the loop!
                }
                failureOutput = testResult.Output
            }
        }

        // IF WE REACH HERE, THE TESTS OR FILE WRITING FAILED
        fmt.Println("❌ ACTION FAILED:")
        fmt.Println(failureOutput)

        if attempt == maxAttempts {
            slog.Error("Agent failed to fix the tests after maximum attempts.", "max_attempts", maxAttempts)
            break
        }

        fmt.Println("🧠 Agent is analyzing the error and attempting a fix...")
        
        // Call the new Fix function, passing the error output back to the AI
		response, err = agent.FixTests(ctx, diff, response.Tests, failureOutput)
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