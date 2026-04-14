package main

import (
	"fmt"
	"log"

	"github.com/blobthebuilder/CICDAgent/internal/agent"
	"github.com/blobthebuilder/CICDAgent/internal/git"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file from the root directory
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file. Make sure it exists and is in the root folder.")
	}

	fmt.Println("🤖 CICD Agent: Analyzing staged changes...")

	// 1. Get the staged changes (git add . must be run first!)
	diff, err := git.GetDiff("staged")
	if err != nil {
		log.Fatalf("Error reading git: %v", err)
	}

	if diff == "" {
		fmt.Println("No staged changes found. Use 'git add' on a file first!")
		return
	}

	// 2. Pass the key explicitly or let the agent package pull it from OS
	review, err := agent.GetReview(diff)
	if err != nil {
		log.Fatalf("AI Error: %v", err)
	}

	fmt.Println("\n--- AI CODE REVIEW ---")
	fmt.Println(review)
}