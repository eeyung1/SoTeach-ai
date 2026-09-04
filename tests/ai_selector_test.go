package tests

import (
	"errors"
	"testing"

	"soteach/ai"
	"soteach/ai/provider"
)

// TestProviderSelectorReturnsGroqForName proves the vendor switch: by name, the
// selector returns the matching concrete ai.AIProvider — so swapping AI
// companies is a config change, never a tutoring-code change (Agent.md §38).
func TestProviderSelectorReturnsGroqForName(t *testing.T) {
	p, err := provider.New("groq", provider.Config{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("expected groq to be selectable, got error: %v", err)
	}

	var _ ai.AIProvider = p
	if _, ok := p.(*provider.Groq); !ok {
		t.Fatalf("expected a *provider.Groq, got %T", p)
	}
}

// TestProviderSelectorRejectsUnknownName confirms an unrecognised provider
// name is a clear error, not a silent fallback.
func TestProviderSelectorRejectsUnknownName(t *testing.T) {
	_, err := provider.New("mystery-company", provider.Config{APIKey: "test-key"})
	if err == nil {
		t.Fatal("expected an unknown provider name to be rejected")
	}
	if !errors.Is(err, provider.ErrUnknownProvider) {
		t.Fatalf("expected ErrUnknownProvider, got %v", err)
	}
}

// TestProviderSelectorRequiresKey confirms a provider without credentials is
// rejected at construction.
func TestProviderSelectorRequiresKey(t *testing.T) {
	_, err := provider.New("groq", provider.Config{})
	if err == nil {
		t.Fatal("expected a missing API key to be rejected")
	}
}
