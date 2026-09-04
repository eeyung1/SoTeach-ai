package tests

import (
	"errors"
	"testing"

	"soteach/session"
	"soteach/tutor"
)

// TestBeginTopicStartsDiagnosisAndReturnsPrompt is the server-side entry into
// the loop (README §6, SELECT_SUBJECT -> SELECT_TOPIC -> DIAGNOSE): when a
// thin client asks the server to begin a topic, the server selects subject
// and topic, owns the diagnostic prompt, and returns it as the first thing
// the learner should respond to. No client decides the wording.
func TestBeginTopicStartsDiagnosisAndReturnsPrompt(t *testing.T) {
	store := session.NewMemoryStore()
	tut := tutor.NewTutor(store)

	outcome, err := tut.BeginTopic("Amaka", "Mathematics", "Addition", "JSS1-3")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	const diagnosticPrompt = "What do you already know about addition?"
	if outcome.Prompt != diagnosticPrompt {
		t.Fatalf("expected the diagnostic prompt %q, got %q", diagnosticPrompt, outcome.Prompt)
	}

	s, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected a session to have been created, got error: %v", loadErr)
	}
	if s.Subject() != "Mathematics" {
		t.Fatalf("expected subject Mathematics, got %q", s.Subject())
	}
	if s.Topic() != "Addition" {
		t.Fatalf("expected topic Addition, got %q", s.Topic())
	}
	if s.State() != session.Diagnosing {
		t.Fatalf("expected state Diagnosing after begin, got %v", s.State())
	}
	if s.DiagnosisComplete() {
		t.Fatal("diagnosis must not be complete before the learner responds")
	}
}

// TestBeginTopicResumesMidSessionWithoutRepeatingDiagnosis confirms begin is
// also the resume path (README §6: resuming an unfinished session must not
// repeat the diagnostic stage): beginning a topic the learner already has in
// progress returns the pending question, not a fresh diagnostic prompt, and
// leaves the recorded diagnosis intact.
func TestBeginTopicResumesMidSessionWithoutRepeatingDiagnosis(t *testing.T) {
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
	outcome, err := tut.BeginTopic("Amaka", "Mathematics", "Addition", "JSS1-3")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	const pendingQuestion = "What is 7 + 3?"
	if outcome.Prompt != pendingQuestion {
		t.Fatalf("expected begin to resume with the pending question %q, got %q", pendingQuestion, outcome.Prompt)
	}

	s, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the session to still load, got error: %v", loadErr)
	}
	if !s.DiagnosisComplete() {
		t.Fatal("resuming must not re-open completed diagnosis")
	}
	if s.DiagnosticFinding() != "I can add single digits" {
		t.Fatalf("resuming must not discard the diagnostic finding, got %q", s.DiagnosticFinding())
	}
	if s.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer on resume, got %v", s.State())
	}
}

// TestBeginMasteredTopicDoesNotResetMastery confirms begin on an already
// mastered topic does not destroy that mastery or re-diagnose (README §6
// per-learner progress must survive resumption); it tells the learner the
// topic is already mastered.
func TestBeginMasteredTopicDoesNotResetMastery(t *testing.T) {
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
	original.SubmitAnswer("10")
	original.CheckAnswer()
	if err := original.AskTransferQuestion("What is 5 + 6?", "11"); err != nil {
		t.Fatalf("expected transfer question to be asked, got error: %v", err)
	}
	original.SubmitAnswer("11")
	original.CheckAnswer()
	store.Save(original)

	tut := tutor.NewTutor(store)
	outcome, err := tut.BeginTopic("Amaka", "Mathematics", "Addition", "JSS1-3")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	const alreadyMastered = "You have already mastered Addition."
	if outcome.Prompt != alreadyMastered {
		t.Fatalf("expected %q, got %q", alreadyMastered, outcome.Prompt)
	}

	s, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the session to still load, got error: %v", loadErr)
	}
	if !s.IsMastered() {
		t.Fatal("begin must not reset recorded mastery")
	}
	if s.State() != session.Mastered {
		t.Fatalf("expected state Mastered, got %v", s.State())
	}
}

// TestBeginTopicRejectsUnsupportedTopic confirms begin only opens topics the
// server has content for (README §5, a deliberately small concept set), and
// rejects an unsupported topic without creating a session.
func TestBeginTopicRejectsUnsupportedTopic(t *testing.T) {
	store := session.NewMemoryStore()
	tut := tutor.NewTutor(store)

	_, err := tut.BeginTopic("Amaka", "Mathematics", "Fractions", "JSS1-3")
	if err == nil {
		t.Fatal("expected an unsupported topic to be rejected")
	}
	if !errors.Is(err, tutor.ErrTopicNotYetSupported) {
		t.Fatalf("expected ErrTopicNotYetSupported, got %v", err)
	}

	if _, loadErr := store.Load("Amaka"); loadErr == nil {
		t.Fatal("a rejected begin must not create a session")
	}
}
