package tests

import (
	"errors"
	"testing"

	"soteach/ai"
	"soteach/session"
	"soteach/tutor"
)

// TestDiagnoseWithAIRecordsValidResult confirms the orchestration wiring
// (Agent.md §42): the AI request happens outside the tutoring engine, and
// only a validated result reaches Session, via the existing
// RecordDiagnosticResult boundary.
func TestDiagnoseWithAIRecordsValidResult(t *testing.T) {
	s := session.New("Amaka")
	s.SelectSubject("Mathematics")
	s.SelectTopic("Addition")
	s.StartDiagnosis("What do you already know about addition?")

	want := ai.DiagnosticResult{
		Concept:           "Addition",
		KnownSkills:       []string{"single-digit addition"},
		Gaps:              []string{"regrouping"},
		Confidence:        0.8,
		RecommendedAction: "teach",
	}
	provider := &ai.FakeAIProvider{DiagnoseResult: want}

	if err := tutor.DiagnoseWithAI(s, provider, "Addition", "I add single digits but get confused when it carries over"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !s.DiagnosisComplete() {
		t.Fatal("expected diagnosis to be marked complete")
	}
	if got := s.LastDiagnosticResult(); got.RecommendedAction != "teach" {
		t.Fatalf("expected stored RecommendedAction %q, got %q", "teach", got.RecommendedAction)
	}
}

// TestDiagnoseWithAIProviderFailureLeavesSessionUntouched confirms
// Agent.md §41 ("AI failure is a normal system condition") and §42 ("a
// provider failure must not corrupt learner state"): a provider error
// must be returned, and the session must remain exactly as it was.
func TestDiagnoseWithAIProviderFailureLeavesSessionUntouched(t *testing.T) {
	s := session.New("Amaka")
	s.SelectSubject("Mathematics")
	s.SelectTopic("Addition")
	s.StartDiagnosis("What do you already know about addition?")

	provider := &ai.FakeAIProvider{DiagnoseErr: errors.New("provider timeout")}

	err := tutor.DiagnoseWithAI(s, provider, "Addition", "I add single digits but get confused when it carries over")
	if err == nil {
		t.Fatal("expected the provider error to be returned")
	}

	if s.DiagnosisComplete() {
		t.Fatal("a provider failure must not mark diagnosis complete")
	}
	if got := s.LastDiagnosticResult(); got.Concept != "" {
		t.Fatalf("a provider failure must not store a diagnostic result, got %+v", got)
	}
}

// TestDiagnoseWithAIRejectsMalformedResult confirms a malformed
// AI-produced result still goes through the same validation as any other
// DiagnosticResult (Agent.md §40) — the orchestration function does not
// bypass RecordDiagnosticResult's validation.
func TestDiagnoseWithAIRejectsMalformedResult(t *testing.T) {
	s := session.New("Amaka")
	s.SelectSubject("Mathematics")
	s.SelectTopic("Addition")
	s.StartDiagnosis("What do you already know about addition?")

	malformed := ai.DiagnosticResult{
		// Concept deliberately omitted.
		RecommendedAction: "teach",
		Confidence:        0.8,
	}
	provider := &ai.FakeAIProvider{DiagnoseResult: malformed}

	err := tutor.DiagnoseWithAI(s, provider, "Addition", "some response")
	if err == nil {
		t.Fatal("expected a malformed result to be rejected")
	}

	if s.DiagnosisComplete() {
		t.Fatal("a rejected result must not mark diagnosis complete")
	}
}
