package main

import (
	"fmt"
	"log"
	"os"

	"github.com/blobthebuilder/CICDAgent/internal/agent"
	"github.com/blobthebuilder/CICDAgent/internal/git"
	"github.com/blobthebuilder/CICDAgent/internal/tools" // <-- Add your tools package
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file. Make sure it exists in the root folder.")
	}

	// Default to "last-commit" mode, but allow overriding from command-line argument
	diffMode := "last-commit"
	if len(os.Args) > 1 {
		diffMode = os.Args[1]
	}

	fmt.Printf("🤖 CICD Agent: Analyzing changes in '%s' mode...\n", diffMode)

	diff, err := git.GetDiff(diffMode)
	if err != nil {
		log.Fatalf("Error reading git: %v", err)
	}

	if diff == "" {
		fmt.Println("No changes found to analyze in the selected mode.")
		return
	}

	// 1. Get the Review and Code from the Agent
	response, err := agent.GetAction(diff)
	if err != nil {
		log.Fatalf("AI Error: %v", err)
	}

	fmt.Println("\n--- 📝 AI CODE REVIEW ---")
	fmt.Println(response.Review)

	// 2. Use your File Tool to save the test
	if response.TestFile != "" && response.TestCode != "" {
		fmt.Printf("\n--- 🛠️ GENERATING TEST: %s ---\n", response.TestFile)
		
		// Call your custom tool here
		writtenPath, err := tools.WriteTestFile(response.TestFile, response.TestCode)
		if err != nil {
			log.Fatalf("Failed to save test file via tool: %v", err)
		}
		
		fmt.Printf("✅ Saved test file to %s\n", writtenPath)
		fmt.Println("💡 Run 'go test ./...' to see if it passes!")
	} else {
		fmt.Println("\nNo test code generated for this diff.")
	}
}