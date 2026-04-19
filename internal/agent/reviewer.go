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

	"github.com/blobthebuilder/CICDAgent/internal/tools"
	"google.golang.org/genai"
)

// GeneratedTest holds the path and code for a single generated test file.
type GeneratedTest struct {
	FileName string `json:"file_name"`
	Imports  string `json:"imports"`
	Code     string `json:"code"`
}

// AgentResponse holds the structured output from the LLM, including a list of tests.
type AgentResponse struct {
	Review         string          `json:"review"`
	Tests          []GeneratedTest `json:"tests"`

	// NEW: Branching fields for the Fix loop
	CodeBugFound   bool   `json:"code_bug_found"`
	BugExplanation string `json:"bug_explanation"`
}

// callGeminiAPI sends a prompt to the Gemini API with a specific JSON schema for the response.
func callGeminiAPI(ctx context.Context, prompt string, schema *genai.Schema) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY not found in environment")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create genai client: %w", err)
	}

	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-2.0-flash"
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   schema,
	}

	result, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), config)
	if err != nil {
		return "", fmt.Errorf("AI generation failed: %w", err)
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
			// Access the Text field directly. No type assertion needed.
			text := result.Candidates[0].Content.Parts[0].Text
			
			if text != "" {
				return text, nil
			}
			return "", fmt.Errorf("AI response was empty")
		}
		
		return "", fmt.Errorf("AI returned no candidates or parts")
	}

// getActionFromGemini attempts to get a response from the Gemini API.
func getActionFromGemini(ctx context.Context, prompt string) (string, error) {
	// --- NEW: FORCE STRICT JSON OUTPUT ---
	schema := &genai.Schema{
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
							Description: "The full relative path for the test file (e.g., 'internal/tools/misc_test.go').",
						},
						"imports": {
							Type:        genai.TypeString,
							Description: "A string containing any new import paths (e.g., `\"fmt\"\\n\"testing\"`) that the generated test functions need. Can be empty.",
						},
						"code": {
							Type:        genai.TypeString,
							Description: "A string containing only the new, complete, and compilable Go test functions to be added.",
						},
					},
					Required: []string{"file_name", "imports", "code"},
				},
			},
		},
		Required: []string{"review", "tests"},
	}

	return callGeminiAPI(ctx, prompt, schema)
}

// callLocalModel sends the prompt to a local, OpenAI-compatible LLM endpoint.
func callLocalModel(ctx context.Context, prompt string) (string, error) {
	endpoint := os.Getenv("LOCAL_LLM_ENDPOINT")
	if endpoint == "" {
		return "", fmt.Errorf("LOCAL_LLM_ENDPOINT not set in environment, and Gemini failed")
	}

	modelName := os.Getenv("LOCAL_LLM_MODEL")
	if modelName == "" {
		modelName = "llama3" // Default model
	}

	// This payload is typical for OpenAI-compatible APIs like Ollama.
	// The `response_format` key is removed for broader compatibility, as not all local models support it.
	// We will rely on prompt instructions and the robust JSON parsing in `parseAgentResponse`.
	payload := map[string]interface{}{
		"model":    modelName,
		"messages": []map[string]string{{"role": "user", "content": prompt}},
		"stream":   false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal local LLM payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(body))
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

// getTestFileContext finds corresponding test files for changed files and extracts their AST info.
func getTestFileContext(diff string) string {
	changedFiles := parseFilenamesFromDiff(diff)
	var astContextBuilder strings.Builder

	foundContext := false
	for _, file := range changedFiles {
		// Construct the test file name
		testFile := strings.TrimSuffix(file, ".go") + "_test.go"

		// Check if the test file exists
		if _, err := os.Stat(testFile); err == nil {
			// File exists, parse it
			info, err := tools.ExtractGoFileInfo(testFile)
			if err != nil {
				slog.Warn("Could not extract AST info for test file", "file", testFile, "error", err)
				continue // Skip files that can't be parsed
			}
			if !foundContext {
				astContextBuilder.WriteString("For context, here are the declarations and function signatures from existing test files:\n\n")
				foundContext = true
			}
			astContextBuilder.WriteString(fmt.Sprintf("--- Existing Test File: %s ---\n%s\n", testFile, info))
		}
	}
	return astContextBuilder.String()
}

// GetAction orchestrates getting a response from an AI model, with a fallback.
func GetAction(ctx context.Context, diff string) (*AgentResponse, error) {
	// Get AST context from existing test files.
	astContext := getTestFileContext(diff)

	prompt := fmt.Sprintf(`
		You are a Senior Software Engineer AI Agent.

		Your task is to review a git diff and generate unit tests.

		RULES:
		1. Summarize the changes. Specifically identify any bugs, security issues, or logic errors in the 'review' field. If you don't find any issues, say "No issues were found."
		2. If a new function is added, write a Go unit test for it.
		3. Use the provided context from existing test files to understand available helpers and test structure. Do not duplicate existing tests.
		4. Place any new, required import paths in the 'imports' field. Each import path should be on its own line (e.g., "fmt").
		5. Place new test functions (e.g., func TestMyFunction(t *testing.T)) in the 'code' field.
		6. The generated code must be complete and compilable.

		%s

		GIT DIFF:
		%s
	`, astContext, diff)

	// Try Gemini first
	rawText, err := getActionFromGemini(ctx, prompt)
	if err == nil {
		slog.Info("Successfully received response from Gemini.")
		return parseAgentResponse(rawText)
	}

	// If Gemini fails, log the error and try the local model
	slog.Warn("Gemini API failed. Falling back to local model.", "error", err)

	rawText, err = callLocalModel(ctx, prompt)
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

// parseFilenamesFromDiff extracts unique Go source file paths from a git diff output.
// It ignores test files.
func parseFilenamesFromDiff(diff string) []string {
	files := make(map[string]struct{})
	lines := strings.Split(diff, "\n")

	for _, line := range lines {
		var path string
		if strings.HasPrefix(line, "--- a/") {
			path = strings.TrimPrefix(line, "--- a/")
		} else if strings.HasPrefix(line, "+++ b/") {
			path = strings.TrimPrefix(line, "+++ b/")
		} else {
			continue
		}

		if path == "dev/null" {
			continue
		}

		// We only care about Go source files, not tests
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files[path] = struct{}{}
		}
	}

	result := make([]string, 0, len(files))
	for f := range files {
		result = append(result, f)
	}
	return result
}

// FixTests asks the AI to fix previously generated tests based on compiler/test errors.
func FixTests(ctx context.Context, diff string, previousTests []GeneratedTest, errorOutput string) (*AgentResponse, error) {
    prevCode := ""
	for _, t := range previousTests {
		prevCode += fmt.Sprintf("\n--- %s ---\n// Imports\n%s\n\n// Code\n%s\n", t.FileName, t.Imports, t.Code)
	}
	astContext := getTestFileContext(diff)

	prompt := fmt.Sprintf(`
        You are a Senior Software Engineer debugging a failing CI pipeline.
        You previously wrote tests for this diff, but they FAILED.

        YOUR TASK: Analyze the error output and determine the root cause.

        PATH A (Test Error): If the error is due to a bad test (e.g., missing import, wrong assertion), set "code_bug_found" to false, leave "bug_explanation" empty, and rewrite the "tests" array with the fixed code.

        PATH B (Code Error): If the test is correct but the SOURCE CODE is broken, set "code_bug_found" to true, provide a detailed "bug_explanation", and leave the "tests" array empty.

		Use the provided file context from existing test files to understand available helpers and test structure.
		When fixing tests, provide new import paths and functions in the 'imports' and 'code' fields respectively.

		%s

		GIT DIFF:
		%s

		PREVIOUS FAILED TESTS:
		%s

		ERROR OUTPUT:
		%s
    `, astContext, diff, prevCode, errorOutput)

	// Try Gemini first
	schema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"code_bug_found":  {Type: genai.TypeBoolean, Description: "True if the source code is broken, false if the test is broken."},
			"bug_explanation": {Type: genai.TypeString, Description: "Explanation of the source code bug (if any)."},
			"review":          {Type: genai.TypeString, Description: "Brief summary of your fix strategy."},
			"tests": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"file_name":    {Type: genai.TypeString},
						"imports": {Type: genai.TypeString, Description: "New import paths needed for the test functions."},
						"code":    {Type: genai.TypeString, Description: "New test functions to add."},
					},
				},
			},
		},
		Required: []string{"code_bug_found", "bug_explanation", "review", "tests"},
	}

	rawText, err := callGeminiAPI(ctx, prompt, schema)
	if err == nil {
		slog.Info("Successfully received fix from Gemini.")
		return parseAgentResponse(rawText)
	}

	// If Gemini fails, log the error and try the local model
	slog.Warn("Gemini API failed to generate a fix. Falling back to local model.", "error", err)

	rawText, err = callLocalModel(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("all models failed to generate a fix. Local model error: %w", err)
	}

	slog.Info("Successfully received fix from local model.")
	return parseAgentResponse(rawText)
}