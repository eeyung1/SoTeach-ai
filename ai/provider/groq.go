// Package provider holds concrete AI vendor adapters that implement the
// ai.AIProvider boundary (Agent.md §38). The tutoring domain never depends on
// a vendor; adding or switching a company is one adapter behind the same
// interface. The current adapter is Groq (OpenAI-compatible chat completions);
// others (Anthropic, Google, ...) slot in identically.
package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"soteach/ai"
)

// GroqConfig configures the Groq adapter. Empty fields get safe defaults
// except APIKey, which is required.
type GroqConfig struct {
	// APIKey is the Groq API key, sent as a Bearer token.
	APIKey string
	// Model is the Groq model id; defaults to llama-3.3-70b-versatile.
	Model string
	// BaseURL defaults to https://api.groq.com/openai/v1.
	BaseURL string
	// BasePath defaults to /chat/completions.
	BasePath string
	// HTTPClient may be injected (e.g. a test double); defaults to a client
	// with a 30s timeout.
	HTTPClient *http.Client
}

// defaultGroqModel is a widely available Groq model. Configurable via Model.
const defaultGroqModel = "llama-3.3-70b-versatile"

// Groq is a concrete AIProvider backed by Groq's OpenAI-compatible
// chat-completions API.
type Groq struct {
	cfg GroqConfig
	cli *http.Client
}

// NewGroq constructs a Groq provider, validating config and applying defaults.
func NewGroq(cfg GroqConfig) (*Groq, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("groq: APIKey is required")
	}
	if cfg.Model == "" {
		cfg.Model = defaultGroqModel
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.groq.com/openai/v1"
	}
	if cfg.BasePath == "" {
		cfg.BasePath = "/chat/completions"
	}
	cli := cfg.HTTPClient
	if cli == nil {
		cli = &http.Client{Timeout: 30 * time.Second}
	}
	return &Groq{cfg: cfg, cli: cli}, nil
}

// systemDiagnosePrompt tells the model to produce exactly the structured JSON
// that ai.DiagnosticResult expects, so output is parsed, not scraped from
// prose (Agent.md §39).
const systemDiagnosePrompt = `You are the diagnosis engine of a diagnostic-first maths tutor for Nigerian school students (Primary 4 through SSS3). Diagnose the learner's current understanding of the concept they are studying.

Return ONLY a single JSON object with exactly these fields and nothing else — no prose, no code fences:
{
  "concept": string,
  "knownSkills": [string],
  "gaps": [string],
  "misconceptions": [string],
  "confidence": number between 0 and 1,
  "recommendedAction": string
}`

// chatRequest is the OpenAI-compatible chat-completions request body.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	MaxTokens int          `json:"max_tokens,omitempty"`
	Temperature float64    `json:"temperature,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Diagnose evaluates the learner's own words about a concept and returns the
// structured DiagnosticResult (Agent.md §37-40). The AI proposes; the engine
// still validates and decides.
func (g *Groq) Diagnose(concept, learnerResponse string) (ai.DiagnosticResult, error) {
	body := chatRequest{
		Model: g.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemDiagnosePrompt},
			{Role: "user", Content: "Concept: " + concept + "\n\nThe learner said: " + learnerResponse},
		},
		Temperature: 0,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return ai.DiagnosticResult{}, err
	}

	req, err := http.NewRequest(http.MethodPost, g.cfg.BaseURL+g.cfg.BasePath, bytes.NewReader(raw))
	if err != nil {
		return ai.DiagnosticResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.cfg.APIKey)

	resp, err := g.cli.Do(req)
	if err != nil {
		return ai.DiagnosticResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return ai.DiagnosticResult{}, fmt.Errorf("groq: provider returned %s: %s", resp.Status, strings.TrimSpace(string(msg)))
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ai.DiagnosticResult{}, fmt.Errorf("groq: decode response: %w", err)
	}
	if out.Error != nil {
		return ai.DiagnosticResult{}, fmt.Errorf("groq: provider error: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return ai.DiagnosticResult{}, errors.New("groq: empty model response")
	}

	text := out.Choices[0].Message.Content
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var result ai.DiagnosticResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return ai.DiagnosticResult{}, fmt.Errorf("groq: model output was not valid JSON: %w", err)
	}
	return result, nil
}
