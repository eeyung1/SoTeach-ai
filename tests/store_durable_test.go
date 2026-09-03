package tests

import (
	"errors"
	"testing"

	"soteach/session"
)

// TestFileStoreRoundTripsAcrossInstances is the durability proof for the
// persistence behavior (Agent.md §20, second half): a session saved through
// one FileStore instance must be loadable — at the correct state — by a
// brand-new FileStore instance over the same directory. That is exactly what
// a server restart does: the resumed session must not lose the learner, the
// pending question, the diagnosis, or the expected answer, and must still
// work when the learner answers.
func TestFileStoreRoundTripsAcrossInstances(t *testing.T) {
	dir := t.TempDir()

	store1, err := session.NewFileStore(dir)
	if err != nil {
		t.Fatalf("expected the file store to open, got error: %v", err)
	}

	original := session.New("Amaka")
	if err := original.SetGradeBand("JSS1-3"); err != nil {
		t.Fatalf("expected grade band to be set, got error: %v", err)
	}
	original.SelectSubject("Mathematics")
	if err := original.SelectTopic("Addition"); err != nil {
		t.Fatalf("expected topic to be selected, got error: %v", err)
	}
	original.StartDiagnosis("What do you already know about addition?")
	original.RecordDiagnosticResponse("I can add single digits")
	if err := original.AskQuestion("What is 7 + 3?", "10"); err != nil {
		t.Fatalf("expected question to be asked, got error: %v", err)
	}
	if err := store1.Save(original); err != nil {
		t.Fatalf("expected Save to succeed, got error: %v", err)
	}

	// A fresh store over the same directory simulates a server restart.
	store2, err := session.NewFileStore(dir)
	if err != nil {
		t.Fatalf("expected the file store to reopen, got error: %v", err)
	}

	resumed, err := store2.Load("Amaka")
	if err != nil {
		t.Fatalf("expected the session to survive a restart, got error: %v", err)
	}

	if resumed.GradeBand() != "JSS1-3" {
		t.Fatalf("expected grade band JSS1-3 to survive, got %q", resumed.GradeBand())
	}
	if resumed.Subject() != "Mathematics" || resumed.Topic() != "Addition" {
		t.Fatalf("expected subject/topic to survive, got %q/%q", resumed.Subject(), resumed.Topic())
	}
	if !resumed.DiagnosisComplete() {
		t.Fatal("a restarted session must not re-open completed diagnosis")
	}
	if resumed.DiagnosticFinding() != "I can add single digits" {
		t.Fatalf("expected the diagnostic finding to survive, got %q", resumed.DiagnosticFinding())
	}
	if resumed.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer after restart, got %v", resumed.State())
	}
	if resumed.CurrentQuestion() != "What is 7 + 3?" {
		t.Fatalf("expected the pending question to survive, got %q", resumed.CurrentQuestion())
	}

	// The expected answer the store persisted must still make the session
	// genuinely answerable after the restart.
	resumed.SubmitAnswer("10")
	if !resumed.CheckAnswer() {
		t.Fatal("10 must be marked correct for 7 + 3 after a restart")
	}
	if resumed.State() != session.AwaitingTransferCheck {
		t.Fatalf("expected AwaitingTransferCheck after correct answer post-restart, got %v", resumed.State())
	}
}

// TestFileStoreLoadUnknownLearnerReturnsError confirms a file store with no
// saved session for a learner reports ErrSessionNotFound, matching MemoryStore.
func TestFileStoreLoadUnknownLearnerReturnsError(t *testing.T) {
	store, err := session.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("expected the file store to open, got error: %v", err)
	}

	_, err = store.Load("Bello")
	if err == nil {
		t.Fatal("expected loading an unsaved learner to error")
	}
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

// TestMemoryStoreAndFileStoreSatisfyStoreContract pins the shared contract the
// tutor and API depend on: a caller that takes a session.Store accepts either
// store, so switching the running server to a durable store needs no tutoring
// or API changes (Agent.md §20).
func TestMemoryStoreAndFileStoreSatisfyStoreContract(t *testing.T) {
	ms := session.NewMemoryStore()
	var _ session.Store = ms

	fs, err := session.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("expected the file store to open, got error: %v", err)
	}
	var _ session.Store = fs
}
