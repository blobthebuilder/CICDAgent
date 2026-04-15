package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"google.golang.org/genai"
)

// GeneratedTest holds the path and code for a single generated test file.
type GeneratedTest struct {
	FileName string
	Code     string
}

// AgentResponse holds the structured output from the LLM, including a list of tests.
type AgentResponse struct {
	Review string
	Tests  []GeneratedTest
}

// getActionFromGemini attempts to get a response from the Gemini API.
func getActionFromGemini(ctx context.Context, prompt string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY not found in environment")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create genai client: %w", err)
	}

	model := "gemini-3-flash-preview"
	result, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("AI generation failed: %w", err)
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		// The response from the AI comes in parts, we need to get the text from the first part.
		return fmt.Sprintf("%s", result.Candidates[0].Content.Parts[0].Text), nil
	}

	return "", fmt.Errorf("the AI returned an empty response")
}

// callLocalModel sends the prompt to a local, OpenAI-compatible LLM endpoint.
func callLocalModel(prompt string) (string, error) {
	endpoint := os.Getenv("LOCAL_LLM_ENDPOINT")
	if endpoint == "" {
		return "", fmt.Errorf("LOCAL_LLM_ENDPOINT not set in environment, and Gemini failed")
	}

	// This payload is typical for OpenAI-compatible APIs like Ollama.
	// You may need to change the model name.
	payload := map[string]interface{}{
		"model":    "llama3",
		"messages": []map[string]string{{"role": "user", "content": prompt}},
		"stream":   false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal local LLM payload: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request for local LLM: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call local LLM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("local LLM returned non-200 status: %s", resp.Status)
	}

	// This struct is simplified for OpenAI/Ollama responses.
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to decode local LLM response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("local LLM returned no choices")
	}

	return apiResp.Choices[0].Message.Content, nil
}

// GetAction orchestrates getting a response from an AI model, with a fallback.
func GetAction(diff string) (*AgentResponse, error) {
	prompt := fmt.Sprintf(`
		You are a Senior Software Engineer AI Agent.

		Review the following git diff.

		TASKS:
		1. Summarize the changes concisely.
		2. Identify bugs, logic errors, or security risks (include file and line numbers).
		3. If a new function is added, write a complete, compilable Go unit test for it, following standard Go practices:
		   - The test file MUST be in the same directory as the file it is testing.
		   - The test file MUST use the same package declaration as the file it is testing.
		   - The test filename MUST be the original filename with a '_test.go' suffix (e.g., a test for 'files.go' should be 'files_test.go').
		   - Since the test is in the same package, you do not need to import the code being tested.

		YOU MUST FORMAT YOUR RESPONSE EXACTLY LIKE THIS. For each test you generate, provide a [FILENAME] block followed by a [TEST_CODE] block.
		[REVIEW]
		<your summary and bug report here>

		[FILENAME]
		<path for test file 1>
		[TEST_CODE]
		<code for test file 1>

		GIT DIFF:
		%s
	`, diff)

	// Try Gemini first
	rawText, err := getActionFromGemini(context.Background(), prompt)
	if err == nil {
		slog.Info("Successfully received response from Gemini.")
		return parseAgentResponse(rawText), nil
	}

	// If Gemini fails, log the error and try the local model
	slog.Warn("Gemini API failed, falling back to local model", "error", err)

	rawText, err = callLocalModel(prompt)
	if err != nil {
		return nil, fmt.Errorf("all models failed. Local model error: %w", err)
	}

	slog.Info("Successfully received response from local model.")
	return parseAgentResponse(rawText), nil
}

// parseAgentResponse breaks the raw LLM string into our struct fields
func parseAgentResponse(raw string) *AgentResponse {
	resp := &AgentResponse{
		Tests: []GeneratedTest{},
	}

	// Normalize newlines to handle different OS conventions
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	// The review is everything before the first occurrence of "[FILENAME]" on a new line.
	// This is more robust than a simple split, as the review text itself might contain the word "[FILENAME]".
	reviewAndFiles := strings.SplitN(raw, "\n[FILENAME]", 2)

	reviewPart := reviewAndFiles[0]
	resp.Review = strings.TrimSpace(strings.ReplaceAll(reviewPart, "[REVIEW]", ""))

	// If there's no second part, it means no "[FILENAME]" marker was found on a new line.
	if len(reviewAndFiles) < 2 {
		return resp
	}

	// The rest of the string contains all the file blocks.
	// Prepend the delimiter we split on so the next split is consistent.
	filesPart := "[FILENAME]" + reviewAndFiles[1]

	// Now, split the filesPart into individual file blocks.
	fileBlocks := strings.Split(filesPart, "[FILENAME]")

	// Process each file block. The first element will be empty, so we skip it.
	for i := 1; i < len(fileBlocks); i++ {
		block := fileBlocks[i]

		// Each file block is split by [TEST_CODE]
		codeParts := strings.SplitN(block, "[TEST_CODE]", 2)
		if len(codeParts) != 2 {
			continue // Malformed block, skip it.
		}

		fileName := strings.TrimSpace(codeParts[0])

		// Clean up the code block
		code := strings.TrimSpace(codeParts[1])
		code = strings.TrimPrefix(code, "```go")
		code = strings.TrimPrefix(code, "```")
		code = strings.TrimSuffix(code, "```")
		code = strings.TrimSpace(code)

		if fileName != "" && code != "" {
			resp.Tests = append(resp.Tests, GeneratedTest{
				FileName: fileName,
				Code:     code,
			})
		}
	}

	return resp
}