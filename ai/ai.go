// Package ai defines the boundary between SoTeach's deterministic
// tutoring engine and any AI provider (Agent.md §36-38). The tutoring
// engine must not depend on a specific vendor; it depends only on
// AIProvider. AI output must be structured (Agent.md §39) rather than
// parsed from arbitrary conversational text, and must be validated
// before it is allowed to change application state (Agent.md §40).
package ai

import "errors"

// DiagnosticResult is the structured output of the DIAGNOSE stage
// (README §2 Stage 1; Agent.md §39). The AI interprets a learner's own
// free-text explanation of what they already know and proposes a
// concept, what the learner already demonstrated, the actual gap, any
// misconceptions, a confidence level, and a recommended next action.
// The application decides what to do with this — the AI proposes, the
// tutoring engine decides (Agent.md §40).
type DiagnosticResult struct {
	Concept           string
	KnownSkills       []string
	Gaps              []string
	Misconceptions    []string
	Confidence        float64
	RecommendedAction string
}

// AIProvider is the boundary the tutoring engine depends on instead of a
// specific vendor (Agent.md §38). Only the capability actually required
// by the current stage is defined here — Diagnose. Other responsibilities
// listed in Agent.md §37 (Explain, GeneratePractice, GenerateVerification,
// AnalyzeReasoning, etc.) are added only when a concrete requirement for
// each exists, not speculatively.
type AIProvider interface {
	Diagnose(concept, learnerResponse string) (DiagnosticResult, error)
}

// ErrMissingConcept is returned when a DiagnosticResult has no Concept —
// a required field (Agent.md §40, "validate required fields").
var ErrMissingConcept = errors.New("diagnostic result is missing a concept")

// ErrMissingRecommendedAction is returned when a DiagnosticResult has no
// RecommendedAction — a required field.
var ErrMissingRecommendedAction = errors.New("diagnostic result is missing a recommended action")

// ErrConfidenceOutOfRange is returned when Confidence falls outside the
// allowed [0, 1] range (Agent.md §40, "validate allowed values").
var ErrConfidenceOutOfRange = errors.New("diagnostic result confidence must be between 0 and 1")

// ValidateDiagnosticResult performs the structural validation required
// before an AI-produced DiagnosticResult may influence application state
// (Agent.md §40). It never silently converts a malformed result into a
// usable one — a validation failure must be rejected outright.
func ValidateDiagnosticResult(r DiagnosticResult) error {
	if r.Concept == "" {
		return ErrMissingConcept
	}
	if r.RecommendedAction == "" {
		return ErrMissingRecommendedAction
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		return ErrConfidenceOutOfRange
	}
	return nil
}
