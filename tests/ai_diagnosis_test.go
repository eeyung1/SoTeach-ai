package tests

import (
	"errors"
	"testing"

	"soteach/ai"
	"soteach/session"
	"soteach/tutor"
)

// TestSubmitDiagnosisWithAIStoresEvaluatedGap proves the funnel's "the model
// evaluates what a learner knows": when the tutor holds an AI provider,
// SubmitDiagnosis runs provider.Diagnose on the learner's explanation,
// stores the validated structured result (Agent.md §39-40), keeps the
// learner's own words verbatim for audit, and still poses the first practice
// question. A gap report is then available.
func TestSubmitDiagnosisWithAIStoresEvaluatedGap(t *testing.T) {
	store := session.NewMemoryStore()
	seedDiagnosingSession(t, store, "Amaka")

	want := ai.DiagnosticResult{
		Concept:           "Addition",
		KnownSkills:       []string{"single-digit addition"},
		Gaps:              []string{"regrouping"},
		Misconceptions:    []string{"treats carrying as optional"},
		Confidence:        0.8,
		RecommendedAction: "teach",
	}
	tut := tutor.NewAITutor(store, &ai.FakeAIProvider{DiagnoseResult: want})

	const explanation = "I add single digits but get confused when it carries over"
	outcome, err := tut.SubmitDiagnosis("Amaka", explanation)
	if err != nil {
		t.Fatalf("expected diagnosis to be accepted, got error: %v", err)
	}
	if outcome.Prompt != "What is 7 + 3?" {
		t.Fatalf("expected the first practice question after diagnosis, got %q", outcome.Prompt)
	}

	s, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the session to load, got error: %v", loadErr)
	}
	if !s.DiagnosisComplete() {
		t.Fatal("expected diagnosis to be complete after the AI evaluation")
	}
	if s.DiagnosticFinding() != explanation {
		t.Fatalf("expected the learner's own words to be kept verbatim, got %q", s.DiagnosticFinding())
	}
	got := s.LastDiagnosticResult()
	if got.Concept != "Addition" {
		t.Fatalf("expected the evaluated concept Addition, got %q", got.Concept)
	}
	if len(got.Gaps) != 1 || got.Gaps[0] != "regrouping" {
		t.Fatalf("expected the evaluated gap regrouping, got %v", got.Gaps)
	}
	if s.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer, got %v", s.State())
	}

	report, reportErr := tut.DiagnosticReport("Amaka")
	if reportErr != nil {
		t.Fatalf("expected a report after diagnosis, got error: %v", reportErr)
	}
	if report.Concept != "Addition" {
		t.Fatalf("expected report concept Addition, got %q", report.Concept)
	}
	if len(report.Gaps) != 1 || report.Gaps[0] != "regrouping" {
		t.Fatalf("expected report gap regrouping, got %v", report.Gaps)
	}
	if report.Finding != explanation {
		t.Fatalf("expected the report to carry the learner's own words, got %q", report.Finding)
	}
}

// TestSubmitDiagnosisAIFailureLeavesSessionOpen confirms Agent.md §41: a
// provider failure is a normal condition — it must not corrupt or close the
// session, which stays recoverable and can be retried once the provider
// recovers.
func TestSubmitDiagnosisAIFailureLeavesSessionOpen(t *testing.T) {
	store := session.NewMemoryStore()
	seedDiagnosingSession(t, store, "Amaka")

	tut := tutor.NewAITutor(store, &ai.FakeAIProvider{DiagnoseErr: errors.New("provider timeout")})
	if _, err := tut.SubmitDiagnosis("Amaka", "I add single digits"); err == nil {
		t.Fatal("expected the provider error to be returned")
	}

	s, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the session to load, got error: %v", loadErr)
	}
	if s.DiagnosisComplete() {
		t.Fatal("a provider failure must not mark diagnosis complete")
	}

	// The learner can retry once the provider recovers.
	recovered := ai.DiagnosticResult{
		Concept:           "Addition",
		Gaps:              []string{"regrouping"},
		Confidence:        0.7,
		RecommendedAction: "teach",
	}
	if _, err := tutor.NewAITutor(store, &ai.FakeAIProvider{DiagnoseResult: recovered}).SubmitDiagnosis("Amaka", "I add single digits"); err != nil {
		t.Fatalf("expected a retry to succeed after the provider recovers, got error: %v", err)
	}
}

// TestDiagnosticReportRejectedBeforeDiagnosisComplete confirms a report is
// only meaningful once a diagnosis actually exists.
func TestDiagnosticReportRejectedBeforeDiagnosisComplete(t *testing.T) {
	store := session.NewMemoryStore()
	seedDiagnosingSession(t, store, "Amaka")

	tut := tutor.NewTutor(store)
	_, err := tut.DiagnosticReport("Amaka")
	if err == nil {
		t.Fatal("expected a report to be rejected before diagnosis is complete")
	}
	if !errors.Is(err, tutor.ErrNoDiagnosisYet) {
		t.Fatalf("expected ErrNoDiagnosisYet, got %v", err)
	}
}

// TestDiagnosticReportWithoutAIReportsVerbatimFinding documents the
// no-provider case: the report still carries the learner's own words and
// topic, with no structured evaluation fields.
func TestDiagnosticReportWithoutAIReportsVerbatimFinding(t *testing.T) {
	store := session.NewMemoryStore()
	seedDiagnosingSession(t, store, "Amaka")

	tut := tutor.NewTutor(store)
	if _, err := tut.SubmitDiagnosis("Amaka", "I can add single digits"); err != nil {
		t.Fatalf("expected diagnosis to be accepted, got error: %v", err)
	}

	report, err := tut.DiagnosticReport("Amaka")
	if err != nil {
		t.Fatalf("expected a report after diagnosis, got error: %v", err)
	}
	if report.Finding != "I can add single digits" {
		t.Fatalf("expected the verbatim finding, got %q", report.Finding)
	}
	if report.Topic != "Addition" {
		t.Fatalf("expected topic Addition, got %q", report.Topic)
	}
	if report.Concept != "" {
		t.Fatalf("expected no concept without an AI evaluation, got %q", report.Concept)
	}
	if len(report.Gaps) != 0 {
		t.Fatalf("expected no gaps without an AI evaluation, got %v", report.Gaps)
	}
}
