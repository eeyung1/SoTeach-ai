package provider

import (
	"errors"

	"soteach/ai"
)

// ErrUnknownProvider is returned when New is asked for a provider name that
// has no adapter registered.
var ErrUnknownProvider = errors.New("no provider adapter for that name")

// Config carries the credentials and options shared by every adapter.
// Provider-specific options stay on that adapter's own config struct.
type Config struct {
	APIKey string
	Model  string
}

// New returns the concrete ai.AIProvider registered under name, or
// ErrUnknownProvider. Adding another AI company is registering one more case
// here and writing its adapter — the tutoring domain never changes
// (Agent.md §38). This is the switch knob the product architecture promises.
func New(name string, cfg Config) (ai.AIProvider, error) {
	switch name {
	case "groq":
		return NewGroq(GroqConfig{APIKey: cfg.APIKey, Model: cfg.Model})
	default:
		return nil, ErrUnknownProvider
	}
}
