package agent

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
)

func GetReview(diff string) (string, error) {
	ctx := context.Background()
	
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY not found in environment")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create genai client: %v", err)
	}

	// Gemini 3 Flash is perfect for high-speed code analysis
	model := "gemini-3-flash-preview" 
	
	prompt := fmt.Sprintf(`
		You are a Senior Software Engineer AI Agent. 
		Review the following git diff for a project.
		
		TASKS:
		1. Summarize the changes concisely.
		2. Identify potential bugs, logic errors, or security risks.
		3. Check for coding standards and best practices.
		4. If a new function is added, suggest a specific test case.
		
		For specific bugs, problems, or suggested changes, give the file and line number that you are referring to.
		GIT DIFF:
		%s
	`, diff)

	result, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("AI generation failed: %v", err)
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		// Extract the text part specifically
		return fmt.Sprintf("%v", result.Candidates[0].Content.Parts[0].Text), nil
	}

	return "The AI analyzed the code but provided an empty response.", nil
}