package tests

import (
	"errors"
	"testing"

	"soteach/session"
	"soteach/tutor"
)

// TestSessionExposesOnlyDefinedGradeBands confirms the server can tell a
// client which grade bands it accepts (README §4: exactly Primary 4-6,
// JSS1-3, SSS1-3) — the single source the picker and begin validation use.
func TestSessionExposesOnlyDefinedGradeBands(t *testing.T) {
	bands := session.GradeBands()
	if len(bands) != 3 {
		t.Fatalf("expected exactly three grade bands, got %v", bands)
	}

	want := map[string]bool{"Primary 4-6": true, "JSS1-3": true, "SSS1-3": true}
	for _, b := range bands {
		if !want[b] {
			t.Fatalf("unexpected grade band %q in %v", b, bands)
		}
	}
}

// seedAwaitingQuestionSession saves a session for the learner that has passed
// diagnosis and is waiting for an answer to "What is 7 + 3?", ready for
// grade-band capture.
func seedAwaitingQuestionSession(t *testing.T, store *session.MemoryStore, learner string) {
	t.Helper()
	s := session.New(learner)
	s.SelectSubject("Mathematics")
	if err := s.SelectTopic("Addition"); err != nil {
		t.Fatalf("expected topic to be selected, got error: %v", err)
	}
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I can add single digits")
	if err := s.AskQuestion("What is 7 + 3?", "10"); err != nil {
		t.Fatalf("expected question to be asked, got error: %v", err)
	}
	store.Save(s)
}

// TestTutorSetGradeBandStoresBandOnSavedSession proves grade band is captured
// on the learner's durable session (README §10 per-learner data; the picker's
// purpose): after SetGradeBand, a fresh load of the same session reports the
// band and is unaffected otherwise — still mid-loop with its question pending.
func TestTutorSetGradeBandStoresBandOnSavedSession(t *testing.T) {
	store := session.NewMemoryStore()
	seedAwaitingQuestionSession(t, store, "Amaka")

	tut := tutor.NewTutor(store)
	if err := tut.SetGradeBand("Amaka", "JSS1-3"); err != nil {
		t.Fatalf("expected grade band to be set, got error: %v", err)
	}

	s, err := store.Load("Amaka")
	if err != nil {
		t.Fatalf("expected the session to load, got error: %v", err)
	}
	if s.GradeBand() != "JSS1-3" {
		t.Fatalf("expected grade band JSS1-3 to be stored, got %q", s.GradeBand())
	}
	if s.State() != session.WaitForAnswer {
		t.Fatalf("setting grade band must not disturb the session, got %v", s.State())
	}
	if s.CurrentQuestion() != "What is 7 + 3?" {
		t.Fatalf("setting grade band must not disturb the pending question, got %q", s.CurrentQuestion())
	}
}

// TestTutorSetGradeBandRejectsInvalidBand confirms an undefined band is
// rejected and does not overwrite an existing one (README §4 is a safety
// requirement, not cosmetic).
func TestTutorSetGradeBandRejectsInvalidBand(t *testing.T) {
	store := session.NewMemoryStore()
	seedAwaitingQuestionSession(t, store, "Amaka")

	tut := tutor.NewTutor(store)
	if err := tut.SetGradeBand("Amaka", "JSS1-3"); err != nil {
		t.Fatalf("expected a valid band to be accepted, got error: %v", err)
	}

	err := tut.SetGradeBand("Amaka", "Grade 9")
	if err == nil {
		t.Fatal("expected an undefined grade band to be rejected")
	}
	if !errors.Is(err, session.ErrInvalidGradeBand) {
		t.Fatalf("expected ErrInvalidGradeBand, got %v", err)
	}

	s, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the session to load, got error: %v", loadErr)
	}
	if s.GradeBand() != "JSS1-3" {
		t.Fatalf("a rejected band must not overwrite the existing one, got %q", s.GradeBand())
	}
}

// TestTutorSetGradeBandUnknownLearnerReturnsError confirms a band cannot be
// set for a learner with no begun session (grade band belongs to a learner who
// has started, in the current model).
func TestTutorSetGradeBandUnknownLearnerReturnsError(t *testing.T) {
	store := session.NewMemoryStore()
	tut := tutor.NewTutor(store)

	err := tut.SetGradeBand("Bello", "JSS1-3")
	if err == nil {
		t.Fatal("expected setting a grade band for an unknown learner to error")
	}
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}
