package tests

import (
	"testing"

	"soteach/session"
	"soteach/tutor"
)

// TestSubmitDiagnosisAsksFirstPracticeQuestion proves the engine owns the
// Diagnose -> Practice transition (README §2; workingReadme.md §3; Agent.md
// §11): when a learner provides their diagnostic explanation, the server
// records it and — without a client choosing the next step — poses the first
// practice question itself, sourced from server-side content. The client
// sends only the learner's words and receives the next prompt to show.
func TestSubmitDiagnosisAsksFirstPracticeQuestion(t *testing.T) {
	store := session.NewMemoryStore()

	original := session.New("Amaka")
	original.SelectSubject("Mathematics")
	if err := original.SelectTopic("Addition"); err != nil {
		t.Fatalf("expected topic to be selected, got error: %v", err)
	}
	original.StartDiagnosis("What do you already know about addition?")
	store.Save(original)

	tut := tutor.NewTutor(store)
	outcome, err := tut.SubmitDiagnosis("Amaka", "I can add single digits")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	const firstQuestion = "What is 7 + 3?"
	if outcome.Prompt != firstQuestion {
		t.Fatalf("expected the tutor to ask %q, got %q", firstQuestion, outcome.Prompt)
	}

	resumed, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the updated session to be saved, got error: %v", loadErr)
	}
	if !resumed.DiagnosisComplete() {
		t.Fatal("diagnosis should be complete after the explanation is recorded")
	}
	if resumed.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer after the first question, got %v", resumed.State())
	}
	if resumed.CurrentQuestion() != firstQuestion {
		t.Fatalf("expected pending question %q, got %q", firstQuestion, resumed.CurrentQuestion())
	}

	// The expected answer the server stored must make the question genuinely
	// answerable, deterministically, by the engine — not by the client.
	resumed.SubmitAnswer("10")
	if !resumed.CheckAnswer() {
		t.Fatal("10 must be marked correct for the tutor's first question")
	}
	if resumed.State() != session.AwaitingTransferCheck {
		t.Fatalf("expected AwaitingTransferCheck after a correct answer, got %v", resumed.State())
	}
}

// TestSubmitDiagnosisRejectedWhenSessionPastDiagnosis confirms the tutor
// refuses to re-accept a diagnostic explanation once a session has moved past
// the diagnostic stage (README §6: resuming must not repeat the diagnostic
// stage). It must error and leave the session exactly as it was.
func TestSubmitDiagnosisRejectedWhenSessionPastDiagnosis(t *testing.T) {
	store := session.NewMemoryStore()

	original := session.New("Amaka")
	original.SelectSubject("Mathematics")
	if err := original.SelectTopic("Addition"); err != nil {
		t.Fatalf("expected topic to be selected, got error: %v", err)
	}
	original.StartDiagnosis("What do you already know about addition?")
	original.RecordDiagnosticResponse("I can add single digits")
	if err := original.AskQuestion("What is 7 + 3?", "10"); err != nil {
		t.Fatalf("expected question to be asked, got error: %v", err)
	}
	store.Save(original)

	tut := tutor.NewTutor(store)
	_, err := tut.SubmitDiagnosis("Amaka", "Actually, I know nothing")
	if err == nil {
		t.Fatal("expected SubmitDiagnosis to be rejected once the session is past diagnosis")
	}

	resumed, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the session to still be loadable, got error: %v", loadErr)
	}
	if resumed.State() != session.WaitForAnswer {
		t.Fatalf("the session must be unchanged; expected WaitForAnswer, got %v", resumed.State())
	}
	if resumed.CurrentQuestion() != "What is 7 + 3?" {
		t.Fatalf("the pending question must be unchanged, got %q", resumed.CurrentQuestion())
	}
	if resumed.DiagnosticFinding() != "I can add single digits" {
		t.Fatalf("the recorded diagnosis must be unchanged, got %q", resumed.DiagnosticFinding())
	}
}
