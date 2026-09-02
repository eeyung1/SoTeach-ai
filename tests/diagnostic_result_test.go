package tests

import (
	"testing"

	"soteach/ai"
	"soteach/session"
)

// TestRecordDiagnosticResultMarksDiagnosisComplete confirms the
// AI-validated diagnosis path (Agent.md §40: AI proposes, the
// application decides) marks diagnosis complete and stores the
// structured result, and that the session behaves as fully diagnosed
// afterward (AskQuestion succeeds).
func TestRecordDiagnosticResultMarksDiagnosisComplete(t *testing.T) {
	s := session.New("Amaka")
	s.SelectSubject("Mathematics")
	s.SelectTopic("Addition")
	s.StartDiagnosis("What do you already know about addition?")

	result := ai.DiagnosticResult{
		Concept:           "Addition",
		KnownSkills:       []string{"single-digit addition"},
		Gaps:              []string{"regrouping"},
		Misconceptions:    []string{"treats carrying as optional"},
		Confidence:        0.8,
		RecommendedAction: "teach",
	}

	if err := s.RecordDiagnosticResult(result); err != nil {
		t.Fatalf("expected a well-formed result to be accepted, got error: %v", err)
	}

	if !s.DiagnosisComplete() {
		t.Fatal("expected diagnosis to be marked complete")
	}

	if got := s.LastDiagnosticResult(); got.RecommendedAction != "teach" {
		t.Fatalf("expected stored RecommendedAction %q, got %q", "teach", got.RecommendedAction)
	}

	if err := s.AskQuestion("What is 7 + 3?", "10"); err != nil {
		t.Fatalf("expected AskQuestion to succeed after diagnosis, got error: %v", err)
	}
}

// TestRecordDiagnosticResultRejectsInvalidResult confirms Agent.md §40's
// "never silently convert malformed AI output into a successful
// tutoring action": an invalid DiagnosticResult must be rejected, and
// must not mark diagnosis complete or change any session state.
func TestRecordDiagnosticResultRejectsInvalidResult(t *testing.T) {
	s := session.New("Amaka")
	s.SelectSubject("Mathematics")
	s.SelectTopic("Addition")
	s.StartDiagnosis("What do you already know about addition?")

	malformed := ai.DiagnosticResult{
		// Concept deliberately omitted.
		RecommendedAction: "teach",
		Confidence:        0.8,
	}

	err := s.RecordDiagnosticResult(malformed)
	if err == nil {
		t.Fatal("expected a malformed result to be rejected")
	}

	if s.DiagnosisComplete() {
		t.Fatal("a rejected result must not mark diagnosis complete")
	}

	if got := s.LastDiagnosticResult(); got.Concept != "" || got.RecommendedAction != "" {
		t.Fatalf("a rejected result must not be stored, got %+v", got)
	}

	if err := s.AskQuestion("What is 7 + 3?", "10"); err == nil {
		t.Fatal("AskQuestion must still be rejected since diagnosis was not actually completed")
	}
}

// TestRecordDiagnosticResultIsResetBySubjectSwitch confirms the stored
// structured result is subject/topic-scoped state, cleared the same way
// diagnosticFinding is (Blueprint lesson #9) rather than leaking into a
// newly selected subject.
func TestRecordDiagnosticResultIsResetBySubjectSwitch(t *testing.T) {
	s := session.New("Amaka")
	s.SelectSubject("Mathematics")
	s.SelectTopic("Addition")
	s.StartDiagnosis("What do you already know about addition?")

	result := ai.DiagnosticResult{
		Concept:           "Addition",
		Confidence:        0.8,
		RecommendedAction: "teach",
	}
	if err := s.RecordDiagnosticResult(result); err != nil {
		t.Fatalf("expected a well-formed result to be accepted, got error: %v", err)
	}

	s.SelectSubject("Christian Religious Studies")

	if got := s.LastDiagnosticResult(); got.Concept != "" {
		t.Fatalf("switching subjects must clear the stored diagnostic result, got %+v", got)
	}
}
