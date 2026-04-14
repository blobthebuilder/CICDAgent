package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/blobthebuilder/CICDAgent/internal/git"
)

func main() {
	fmt.Println("🤖 CICD Agent: Analyzing changes...")

	// Get diff of the last commit
	diff, err := git.GetDiff("HEAD~1")
	if err != nil {
		log.Fatalf("Failed to read git diff: %v", err)
	}

	if len(strings.TrimSpace(diff)) == 0 {
		fmt.Println("No changes detected since the last commit.")
		return
	}

	fmt.Println("--- Current Diff ---")
	fmt.Println(diff)
	
	// NEXT STEP: Send this 'diff' string to the LLM
}