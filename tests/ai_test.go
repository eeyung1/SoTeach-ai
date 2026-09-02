package tests

import (
	"testing"

	"soteach/ai"
)

// TestFakeAIProviderReturnsConfiguredDiagnosticResult confirms the test
// double behaves as a stand-in AIProvider (README §36-38, Agent.md §43):
// domain tests must be able to exercise Diagnose without a live model,
// network access, or credentials.
func TestFakeAIProviderReturnsConfiguredDiagnosticResult(t *testing.T) {
	want := ai.DiagnosticResult{
		Concept:           "Addition",
		KnownSkills:       []string{"single-digit addition"},
		Gaps:              []string{"regrouping"},
		Misconceptions:    []string{"treats carrying as optional"},
		Confidence:        0.8,
		RecommendedAction: "teach",
	}
	fake := &ai.FakeAIProvider{DiagnoseResult: want}

	got, err := fake.Diagnose("Addition", "I add single digits but get confused when it carries over")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Concept != want.Concept {
		t.Fatalf("expected Concept %q, got %q", want.Concept, got.Concept)
	}
	if got.RecommendedAction != want.RecommendedAction {
		t.Fatalf("expected RecommendedAction %q, got %q", want.RecommendedAction, got.RecommendedAction)
	}
	if got.Confidence != want.Confidence {
		t.Fatalf("expected Confidence %v, got %v", want.Confidence, got.Confidence)
	}
}

// TestValidateDiagnosticResultRejectsEmptyConcept is a direct test of
// Agent.md §40: AI output must be validated before it can influence
// application state. A missing concept is a missing required field.
func TestValidateDiagnosticResultRejectsEmptyConcept(t *testing.T) {
	r := ai.DiagnosticResult{RecommendedAction: "teach", Confidence: 0.5}
	if err := ai.ValidateDiagnosticResult(r); err == nil {
		t.Fatal("expected a result with an empty Concept to be rejected")
	}
}

// TestValidateDiagnosticResultRejectsEmptyRecommendedAction confirms the
// same required-field validation for RecommendedAction.
func TestValidateDiagnosticResultRejectsEmptyRecommendedAction(t *testing.T) {
	r := ai.DiagnosticResult{Concept: "Addition", Confidence: 0.5}
	if err := ai.ValidateDiagnosticResult(r); err == nil {
		t.Fatal("expected a result with an empty RecommendedAction to be rejected")
	}
}

// TestValidateDiagnosticResultRejectsConfidenceOutOfRange confirms
// Confidence is validated as an allowed value (Agent.md §40), not just
// present: it must fall within [0, 1].
func TestValidateDiagnosticResultRejectsConfidenceOutOfRange(t *testing.T) {
	tooHigh := ai.DiagnosticResult{Concept: "Addition", RecommendedAction: "teach", Confidence: 1.5}
	if err := ai.ValidateDiagnosticResult(tooHigh); err == nil {
		t.Fatal("expected Confidence above 1 to be rejected")
	}

	negative := ai.DiagnosticResult{Concept: "Addition", RecommendedAction: "teach", Confidence: -0.1}
	if err := ai.ValidateDiagnosticResult(negative); err == nil {
		t.Fatal("expected negative Confidence to be rejected")
	}
}

// TestValidateDiagnosticResultAcceptsWellFormedResult confirms a
// correctly structured result is not rejected.
func TestValidateDiagnosticResultAcceptsWellFormedResult(t *testing.T) {
	r := ai.DiagnosticResult{
		Concept:           "Addition",
		KnownSkills:       []string{"single-digit addition"},
		Gaps:              []string{"regrouping"},
		Confidence:        0.75,
		RecommendedAction: "teach",
	}
	if err := ai.ValidateDiagnosticResult(r); err != nil {
		t.Fatalf("expected a well-formed result to be accepted, got error: %v", err)
	}
}
