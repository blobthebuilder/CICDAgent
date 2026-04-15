package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/blobthebuilder/CICDAgent/internal/agent"
	"github.com/blobthebuilder/CICDAgent/internal/git"
	"github.com/blobthebuilder/CICDAgent/internal/tools" // <-- Add your tools package
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		slog.Warn("Error loading .env file. Make sure it exists in the root folder.")
	}

	// Default to "last-commit" mode, but allow overriding from command-line argument
	diffMode := "last-commit"
	if len(os.Args) > 1 {
		diffMode = os.Args[1]
	}

	fmt.Printf("🤖 CICD Agent: Analyzing changes in '%s' mode...\n", diffMode)

	diff, err := git.GetDiff(diffMode)
	if err != nil {
		slog.Error("Error reading git", "error", err)
		os.Exit(1)
	}

	if diff == "" {
		fmt.Println("No changes found to analyze in the selected mode.")
		return
	}

	// 1. Get the Review and Code from the Agent
	response, err := agent.GetAction(diff)
	if err != nil {
		slog.Error("AI Error", "error", err)
		os.Exit(1)
	}

	fmt.Println("\n--- 📝 AI CODE REVIEW ---")
	fmt.Println(response.Review)

	// 2. Use your File Tool to save the generated tests
	if len(response.Tests) > 0 {
		fmt.Printf("\n--- 🛠️ GENERATING %d TEST FILE(S) ---\n", len(response.Tests))
		for _, test := range response.Tests {
			fmt.Printf("-> Writing test for %s...\n", test.FileName)
			writtenPath, err := tools.WriteTestFile(test.FileName, test.Code)
			if err != nil {
				slog.Error("Failed to save test file", "file", test.FileName, "error", err)
				os.Exit(1)
			}
			fmt.Printf("✅ Saved test file to %s\n", writtenPath)
		}
		fmt.Println("\n💡 Run 'go test ./...' to see if it passes!")
	} else {
		fmt.Println("\nNo test code generated for this diff.")
	}
}