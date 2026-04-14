package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"google.golang.org/genai"
)

// AgentResponse holds the structured output from the LLM
type AgentResponse struct {
	Review   string
	TestFile string
	TestCode string
}

func GetAction(diff string) (*AgentResponse, error) {
	ctx := context.Background()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not found in environment")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %v", err)
	}

	model := "gemini-3-flash-preview"

	// Notice the strict formatting instructions
	prompt := fmt.Sprintf(`
		You are a Senior Software Engineer AI Agent. 
		Review the following git diff.
		
		TASKS:
		1. Summarize the changes concisely.
		2. Identify bugs, logic errors, or security risks (include file and line numbers).
		3. If a new function is added, write a complete, compilable Go unit test for it.
		
		YOU MUST FORMAT YOUR RESPONSE EXACTLY LIKE THIS:
		[REVIEW]
		<your summary and bug report here>
		[FILENAME]
		<the suggested test filename, e.g., math_utils_test.go>
		[TEST_CODE]
		<the raw go code for the test, without markdown backticks>

		GIT DIFF:
		%s
	`, diff)

	result, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), nil)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %v", err)
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		rawText := fmt.Sprintf("%v", result.Candidates[0].Content.Parts[0].Text)
		return parseAgentResponse(rawText), nil
	}

	return nil, fmt.Errorf("the AI returned an empty response")
}

// parseAgentResponse breaks the raw LLM string into our struct fields
func parseAgentResponse(raw string) *AgentResponse {
	resp := &AgentResponse{}

	// Split the text using the tags we told the AI to generate
	parts := strings.Split(raw, "[FILENAME]")
	if len(parts) == 2 {
		resp.Review = strings.ReplaceAll(strings.TrimSpace(parts[0]), "[REVIEW]", "")

		subParts := strings.Split(parts[1], "[TEST_CODE]")
		if len(subParts) == 2 {
			resp.TestFile = strings.TrimSpace(subParts[0])

			// Clean up the code block just in case the AI added markdown ```go backticks
			code := strings.TrimSpace(subParts[1])
			code = strings.TrimPrefix(code, "```go")
			code = strings.TrimPrefix(code, "```")
			code = strings.TrimSuffix(code, "```")
			
			resp.TestCode = strings.TrimSpace(code)
		}
	} else {
		// Fallback if the AI fails to use the formatting
		resp.Review = raw
	}

	return resp
}