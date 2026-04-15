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
	FileName string `json:"file_name"`
	Code     string `json:"code"`
}

// AgentResponse holds the structured output from the LLM, including a list of tests.
type AgentResponse struct {
	Review string          `json:"review"`
	Tests  []GeneratedTest `json:"tests"`
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

	// --- NEW: FORCE STRICT JSON OUTPUT ---
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"review": {
					Type:        genai.TypeString,
					Description: "A concise summary of the changes, identifying bugs or logic errors.",
				},
				"tests": {
					Type: genai.TypeArray,
					Items: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"file_name": {
								Type:        genai.TypeString,
								Description: "The full relative path for the test file.",
							},
							"code": {
								Type:        genai.TypeString,
								Description: "The complete, compilable Go unit test code.",
							},
						},
						Required: []string{"file_name", "code"},
					},
				},
			},
			Required: []string{"review", "tests"},
		},
	}

	// Pass the 'config' object here instead of 'nil'
	result, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), config)
	if err != nil {
		return "", fmt.Errorf("AI generation failed: %w", err)
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 { 
        text := result.Candidates[0].Content.Parts[0].Text
        
        if text != "" {
            return text, nil
        }
        return "", fmt.Errorf("AI response was empty")
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
		"response_format": map[string]string{"type": "json_object"},
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

        Your task is to review a git diff and generate unit tests.
        
        RULES:
        1. Summarize the changes and identify any bugs or logic errors.
        2. If a new function is added, write a complete, compilable Go unit test for it.
        3. The test MUST be in the same package as the file it is testing.
		4. Include file names and line numbers when relevant in the review.

        GIT DIFF:
        %s
	`, diff)

	// Try Gemini first
	rawText, err := getActionFromGemini(context.Background(), prompt)
	if err == nil {
		slog.Info("Successfully received response from Gemini.")
		return parseAgentResponse(rawText)
	}

	// If Gemini fails, log the error and try the local model
	slog.Warn("Gemini API failed. Falling back to local model.", "error", err)

	rawText, err = callLocalModel(prompt)
	if err != nil {
		return nil, fmt.Errorf("all models failed. Local model error: %w", err)
	}

	slog.Info("Successfully received response from local model.")
	return parseAgentResponse(rawText)
}

// parseAgentResponse breaks the raw LLM string into our struct fields
func parseAgentResponse(rawJSON string) (*AgentResponse, error) {
	resp := &AgentResponse{
		Tests: []GeneratedTest{},
	}

	// The AI may still wrap the JSON in markdown, so we defensively strip it.
	cleanJSON := strings.TrimSpace(rawJSON)
	cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
	cleanJSON = strings.TrimPrefix(cleanJSON, "```")
	cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	cleanJSON = strings.TrimSpace(cleanJSON)

	if err := json.Unmarshal([]byte(cleanJSON), resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w\nRaw Response:\n%s", err, rawJSON)
	}
	
	return resp, nil
}

// FixTests asks the AI to fix previously generated tests based on compiler/test errors.
func FixTests(diff string, previousTests []GeneratedTest, errorOutput string) (*AgentResponse, error) {
    // Convert previous tests to a string so the AI knows what it wrote last time
    prevCode := ""
    for _, t := range previousTests {
        prevCode += fmt.Sprintf("\n--- %s ---\n%s\n", t.FileName, t.Code)
    }

    prompt := fmt.Sprintf(`
        You are a Senior Software Engineer AI Agent debugging a failed CI pipeline.

        You previously wrote tests for this git diff, but they FAILED to compile or pass.
        Analyze the error output and rewrite the tests to fix the issue.
        
        RULES:
        1. Read the ERROR OUTPUT carefully. If it's a missing import, add it. If it's a logic error, fix the assertion.
        2. Return the COMPLETE fixed test file(s). Do not just return the snippet that changed.

        GIT DIFF:
        %s

        PREVIOUS TEST CODE YOU WROTE:
        %s

        ERROR OUTPUT FROM 'go test':
        %s
    `, diff, prevCode, errorOutput)

    // Try Gemini
    rawText, err := getActionFromGemini(context.Background(), prompt)
    if err == nil {
        slog.Info("Successfully received fix from Gemini.")
        return parseAgentResponse(rawText)
    }

    // Fallback to Local
    slog.Warn("Gemini API failed during fix. Falling back to local model.", "error", err)
    rawText, err = callLocalModel(prompt)
    if err != nil {
        return nil, fmt.Errorf("all models failed during fix loop: %w", err)
    }

    return parseAgentResponse(rawText)
}