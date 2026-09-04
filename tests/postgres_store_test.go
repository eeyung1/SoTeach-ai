package tests

import (
	"errors"
	"os"
	"testing"

	"soteach/session"
)

// postgresTestDSN returns the DSN from SOTEACH_TEST_DSN, skipping the test when
// it is not set so ordinary `go test ./...` needs no Postgres (Agent.md §43).
func postgresTestDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("SOTEACH_TEST_DSN")
	if dsn == "" {
		t.Skip("set SOTEACH_TEST_DSN to run the Postgres store tests")
	}
	return dsn
}

// TestPostgresStoreRoundTripsAcrossInstances is the durability proof for the
// PostgreSQL store: a session saved through one PostgresStore is loadable — at
// the correct state — by a fresh PostgresStore over the same database, exactly
// like a server restart, and is still answerable afterwards.
func TestPostgresStoreRoundTripsAcrossInstances(t *testing.T) {
	dsn := postgresTestDSN(t)

	store1, err := session.NewPostgresStore(dsn)
	if err != nil {
		t.Fatalf("expected the postgres store to open, got error: %v", err)
	}
	defer store1.Close()

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

	// A fresh store over the same database simulates a server restart.
	store2, err := session.NewPostgresStore(dsn)
	if err != nil {
		t.Fatalf("expected the postgres store to reopen, got error: %v", err)
	}
	defer store2.Close()

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
	if resumed.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer after restart, got %v", resumed.State())
	}
	if resumed.CurrentQuestion() != "What is 7 + 3?" {
		t.Fatalf("expected the pending question to survive, got %q", resumed.CurrentQuestion())
	}

	resumed.SubmitAnswer("10")
	if !resumed.CheckAnswer() {
		t.Fatal("10 must be marked correct for 7 + 3 after a restart")
	}
}

// TestPostgresStoreLoadUnknownLearnerReturnsError confirms a store with no row
// for a learner reports ErrSessionNotFound, matching the other stores.
func TestPostgresStoreLoadUnknownLearnerReturnsError(t *testing.T) {
	dsn := postgresTestDSN(t)
	store, err := session.NewPostgresStore(dsn)
	if err != nil {
		t.Fatalf("expected the postgres store to open, got error: %v", err)
	}
	defer store.Close()

	_, err = store.Load("Bello")
	if err == nil {
		t.Fatal("expected loading an unsaved learner to error")
	}
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

// TestPostgresStoreSatisfiesStoreContract pins that PostgresStore can be used
// anywhere a session.Store is expected (compile-time check).
func TestPostgresStoreSatisfiesStoreContract(t *testing.T) {
	var _ session.Store = (*session.PostgresStore)(nil)
}
