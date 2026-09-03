package tests

import (
	"testing"

	"soteach/session"
)

// TestSavedSessionResumesWithoutRepeatingDiagnosis is a direct test of the
// session-persistence requirement (Agent.md §20) behind the resume rule in
// README §6 ("must support resuming an unfinished session without repeating
// the diagnostic stage from scratch") and workingReadme.md §9 (a resumed
// session must continue at the correct state step). A Session interrupted
// mid-loop is saved to a store, and loading it back must return a Session
// that continues exactly where it left off — same learner, grade band,
// subject, topic, pending question, and state — with diagnosis still
// complete. It must not lose the pending question, and it must still behave
// correctly once the learner answers it.
func TestSavedSessionResumesWithoutRepeatingDiagnosis(t *testing.T) {
	store := session.NewMemoryStore()

	original := session.New("Amaka")
	if err := original.SetGradeBand("JSS1-3"); err != nil {
		t.Fatalf("expected grade band to be set, got error: %v", err)
	}
	original.SelectSubject("Mathematics")
	if err := original.SelectTopic("Addition"); err != nil {
		t.Fatalf("expected topic to be selected, got error: %v", err)
	}
	original.StartDiagnosis("What do you already know about addition?")
	original.RecordDiagnosticResponse("I know how to add single digits")
	if err := original.AskQuestion("What is 7 + 3?", "10"); err != nil {
		t.Fatalf("expected question to be asked, got error: %v", err)
	}

	store.Save(original)

	resumed, err := store.Load("Amaka")
	if err != nil {
		t.Fatalf("expected to load the saved session, got error: %v", err)
	}

	if resumed.LearnerName() != "Amaka" {
		t.Fatalf("expected learner Amaka to survive save/load, got %q", resumed.LearnerName())
	}
	if resumed.GradeBand() != "JSS1-3" {
		t.Fatalf("expected grade band JSS1-3 to survive save/load, got %q", resumed.GradeBand())
	}
	if resumed.Subject() != "Mathematics" {
		t.Fatalf("expected subject Mathematics to survive save/load, got %q", resumed.Subject())
	}
	if resumed.Topic() != "Addition" {
		t.Fatalf("expected topic Addition to survive save/load, got %q", resumed.Topic())
	}
	if !resumed.DiagnosisComplete() {
		t.Fatal("resuming must not re-open diagnosis that was already complete")
	}
	if resumed.DiagnosticFinding() != "I know how to add single digits" {
		t.Fatalf("expected the diagnostic finding to survive save/load, got %q", resumed.DiagnosticFinding())
	}
	if resumed.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer to survive save/load, got %v", resumed.State())
	}
	if resumed.CurrentQuestion() != "What is 7 + 3?" {
		t.Fatalf("expected the pending question to survive save/load, got %q", resumed.CurrentQuestion())
	}
	if resumed.HasAnswer() {
		t.Fatal("a saved session with no learner answer yet must not gain one on load")
	}

	// The resumed session must still be live: it continues the same loop and
	// still checks answers against the original expected answer.
	resumed.SubmitAnswer("10")
	if !resumed.CheckAnswer() {
		t.Fatal("10 must be marked correct for 7 + 3 after resuming")
	}
	if resumed.State() != session.AwaitingTransferCheck {
		t.Fatalf("expected state AwaitingTransferCheck after correct answer on resumed session, got %v", resumed.State())
	}
}

// TestLoadingUnsavedLearnerReturnsError confirms a store only returns a
// session that was actually saved: loading a learner with no saved session
// must error rather than return a zero-value session the caller could
// mistake for real resumed state.
func TestLoadingUnsavedLearnerReturnsError(t *testing.T) {
	store := session.NewMemoryStore()

	_, err := store.Load("Bello")
	if err == nil {
		t.Fatal("loading a learner with no saved session must return an error")
	}
}

// TestSaveCapturesSnapshotNotLiveSession confirms Save records the Session's
// state at the moment it is called (Agent.md §20; README §6 must not lose
// state): the store must not hold a pointer to the live Session. Mutating
// the original Session after Save — submitting and checking an answer,
// changing the diagnostic finding — must not leak into what Load later
// returns. Without this guarantee a saved session could silently resume from
// a point the learner never reached at save time.
func TestSaveCapturesSnapshotNotLiveSession(t *testing.T) {
	store := session.NewMemoryStore()

	original := session.New("Amaka")
	original.SelectSubject("Mathematics")
	original.SelectTopic("Addition")
	original.StartDiagnosis("What do you already know about addition?")
	original.RecordDiagnosticResponse("I know how to add single digits")
	if err := original.AskQuestion("What is 7 + 3?", "10"); err != nil {
		t.Fatalf("expected question to be asked, got error: %v", err)
	}

	store.Save(original)

	// Mutate the live session after saving: submit a wrong answer, check it,
	// and change the diagnostic finding.
	original.SubmitAnswer("9")
	original.CheckAnswer()
	original.RecordDiagnosticResponse("I actually do not know this at all")

	resumed, err := store.Load("Amaka")
	if err != nil {
		t.Fatalf("expected to load the saved session, got error: %v", err)
	}

	if resumed.State() != session.WaitForAnswer {
		t.Fatalf("post-Save mutations must not change the loaded state; expected WaitForAnswer, got %v", resumed.State())
	}
	if resumed.CurrentQuestion() != "What is 7 + 3?" {
		t.Fatalf("post-Save mutations must not change the loaded question, got %q", resumed.CurrentQuestion())
	}
	if resumed.HasAnswer() {
		t.Fatal("an answer submitted after Save must not appear in the loaded session")
	}
	if resumed.DiagnosticFinding() != "I know how to add single digits" {
		t.Fatalf("a diagnostic finding changed after Save must not leak into the loaded session, got %q", resumed.DiagnosticFinding())
	}

	// The loaded session must still be usable from the point it was saved.
	resumed.SubmitAnswer("10")
	if !resumed.CheckAnswer() {
		t.Fatal("10 must be marked correct for 7 + 3 on the loaded session")
	}
	if resumed.State() != session.AwaitingTransferCheck {
		t.Fatalf("expected state AwaitingTransferCheck after correct answer on loaded session, got %v", resumed.State())
	}
}
