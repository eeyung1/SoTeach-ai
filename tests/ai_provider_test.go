package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"soteach/ai"
	"soteach/ai/provider"
)

// TestGroqDiagnoseParsesStructuredResult proves a real, concrete provider
// (Groq, OpenAI-compatible chat completions) implements the AIProvider
// boundary: Diagnose posts the learner's response to the model with a
// structured-JSON instruction, and parses the returned JSON into a
// ai.DiagnosticResult the engine can validate (Agent.md §38-40). The request
// is captured by a local server — no live network, no API key needed.
func TestGroqDiagnoseParsesStructuredResult(t *testing.T) {
	wantResult := ai.DiagnosticResult{
		Concept:           "Addition",
		KnownSkills:       []string{"single-digit addition"},
		Gaps:              []string{"regrouping"},
		Misconceptions:    []string{"treats carrying as optional"},
		Confidence:        0.8,
		RecommendedAction: "teach",
	}
	raw, err := json.Marshal(wantResult)
	if err != nil {
		t.Fatalf("expected the fixture to marshal, got error: %v", err)
	}

	var captured struct {
		Authorization string
		Body          struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected path /chat/completions, got %s", r.URL.Path)
		}
		captured.Authorization = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&captured.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": string(raw)}},
			},
		})
	}))
	defer srv.Close()

	p, err := provider.NewGroq(provider.GroqConfig{
		APIKey:   "test-key",
		Model:    "llama-3.3-70b-versatile",
		BaseURL:  srv.URL,
		BasePath: "/chat/completions",
	})
	if err != nil {
		t.Fatalf("expected the groq provider to construct, got error: %v", err)
	}

	got, err := p.Diagnose("Addition", "I add single digits but get confused when it carries over")
	if err != nil {
		t.Fatalf("expected Diagnose to succeed, got error: %v", err)
	}

	if captured.Authorization != "Bearer test-key" {
		t.Fatalf("expected Bearer authorization, got %q", captured.Authorization)
	}
	if captured.Body.Model != "llama-3.3-70b-versatile" {
		t.Fatalf("expected the configured model, got %q", captured.Body.Model)
	}
	if len(captured.Body.Messages) != 2 {
		t.Fatalf("expected a system and a user message, got %d", len(captured.Body.Messages))
	}
	if captured.Body.Messages[0].Role != "system" ||
		!strings.Contains(captured.Body.Messages[0].Content, "JSON") {
		t.Fatalf("expected a system instruction asking for JSON, got role=%q content=%q",
			captured.Body.Messages[0].Role, captured.Body.Messages[0].Content)
	}
	if captured.Body.Messages[1].Role != "user" ||
		!strings.Contains(captured.Body.Messages[1].Content, "I add single digits") {
		t.Fatalf("expected the learner's response in the user message, got role=%q content=%q",
			captured.Body.Messages[1].Role, captured.Body.Messages[1].Content)
	}

	if got.Concept != "Addition" {
		t.Fatalf("expected concept Addition, got %q", got.Concept)
	}
	if len(got.Gaps) != 1 || got.Gaps[0] != "regrouping" {
		t.Fatalf("expected gap regrouping, got %v", got.Gaps)
	}
	if got.RecommendedAction != "teach" {
		t.Fatalf("expected recommended action teach, got %q", got.RecommendedAction)
	}
}

// TestGroqDiagnoseParsesResultWrappedInProse confirms a model that ignores the
// "JSON only" instruction and wraps the object in prose is still parsed
// correctly (Agent.md §40 — robust structured-output handling).
func TestGroqDiagnoseParsesResultWrappedInProse(t *testing.T) {
	want := ai.DiagnosticResult{
		Concept:           "Addition",
		Gaps:              []string{"regrouping"},
		Confidence:        0.7,
		RecommendedAction: "teach",
	}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("expected the fixture to marshal, got error: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "Sure! Here is your analysis:\n\n" + string(raw) + "\n\nHope that helps."}},
			},
		})
	}))
	defer srv.Close()

	p, err := provider.NewGroq(provider.GroqConfig{
		APIKey:   "test-key",
		Model:    "llama-3.3-70b-versatile",
		BaseURL:  srv.URL,
		BasePath: "/chat/completions",
	})
	if err != nil {
		t.Fatalf("expected the groq provider to construct, got error: %v", err)
	}

	got, err := p.Diagnose("Addition", "I add single digits but get confused when it carries over")
	if err != nil {
		t.Fatalf("expected Diagnose to succeed despite prose, got error: %v", err)
	}
	if got.Concept != "Addition" || len(got.Gaps) != 1 || got.Gaps[0] != "regrouping" {
		t.Fatalf("expected the wrapped JSON to be parsed, got %+v", got)
	}
}

// TestGroqDiagnoseReturnsProviderError confirms a non-2xx response from the
// model API surfaces as an error (Agent.md §41 — the session stays untouched).
func TestGroqDiagnoseReturnsProviderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"upstream failure"}}`))
	}))
	defer srv.Close()

	p, err := provider.NewGroq(provider.GroqConfig{
		APIKey:   "test-key",
		Model:    "llama-3.3-70b-versatile",
		BaseURL:  srv.URL,
		BasePath: "/chat/completions",
	})
	if err != nil {
		t.Fatalf("expected the groq provider to construct, got error: %v", err)
	}

	if _, err := p.Diagnose("Addition", "I add single digits"); err == nil {
		t.Fatal("expected a provider error to be returned")
	}
}

// TestGroqDiagnoseRejectsMalformedContent confirms a response that is not
// valid JSON is rejected rather than silently parsed into a zero result.
func TestGroqDiagnoseRejectsMalformedContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "definitely not json"}},
			},
		})
	}))
	defer srv.Close()

	p, err := provider.NewGroq(provider.GroqConfig{
		APIKey:   "test-key",
		Model:    "llama-3.3-70b-versatile",
		BaseURL:  srv.URL,
		BasePath: "/chat/completions",
	})
	if err != nil {
		t.Fatalf("expected the groq provider to construct, got error: %v", err)
	}

	if _, err := p.Diagnose("Addition", "I add single digits"); err == nil {
		t.Fatal("expected malformed model output to be rejected")
	}
}
