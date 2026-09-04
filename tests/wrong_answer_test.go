package tests

import (
	"strings"
	"testing"

	"soteach/session"
	"soteach/tutor"
)

// startPracticing drives a fresh learner through begin + diagnosis to a
// pending first practice question (7 + 3), returning the tutor.
func startPracticing(t *testing.T, store *session.MemoryStore, learner string) *tutor.Tutor {
	t.Helper()
	tut := tutor.NewTutor(store)
	if _, err := tut.BeginTopic(learner, "Mathematics", "Addition"); err != nil {
		t.Fatalf("expected begin to succeed, got error: %v", err)
	}
	if _, err := tut.SubmitDiagnosis(learner, "I can add single digits"); err != nil {
		t.Fatalf("expected diagnosis to be accepted, got error: %v", err)
	}
	return tut
}

// TestServerTeachesAfterRepeatedWrongAnswers closes the wrong-answer gap: a
// learner who keeps getting the same gap wrong is eventually taught it
// (README §9: no child left floundering), not endlessly re-asked. One wrong
// answer gets a fresh question; the second consecutive wrong answer triggers
// teaching, and only then a fresh question.
func TestServerTeachesAfterRepeatedWrongAnswers(t *testing.T) {
	store := session.NewMemoryStore()
	tut := startPracticing(t, store, "Amaka")

	const explanation = "Addition means putting two groups together and counting all the items to find the total."

	// First wrong answer: a fresh question for the same gap, no teaching yet.
	first, err := tut.SubmitAnswer("Amaka", "9")
	if err != nil {
		t.Fatalf("expected the wrong answer to be accepted, got error: %v", err)
	}
	if first.Prompt != "What is 4 + 6?" {
		t.Fatalf("expected a fresh question after the first wrong answer, got %q", first.Prompt)
	}
	if strings.Contains(first.Prompt, explanation) {
		t.Fatalf("the first wrong answer must not trigger teaching yet, got %q", first.Prompt)
	}

	// Second consecutive wrong answer: the server teaches the gap, then asks.
	second, err := tut.SubmitAnswer("Amaka", "1")
	if err != nil {
		t.Fatalf("expected the second wrong answer to be accepted, got error: %v", err)
	}
	if !strings.Contains(second.Prompt, explanation) {
		t.Fatalf("expected teaching after repeated wrong answers, got %q", second.Prompt)
	}
	if !strings.Contains(second.Prompt, "?") {
		t.Fatalf("expected a fresh question after the teaching, got %q", second.Prompt)
	}

	s, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the session to load, got error: %v", loadErr)
	}
	if s.LastTeaching() != explanation {
		t.Fatalf("expected the explanation to be recorded for auditability, got %q", s.LastTeaching())
	}
	if s.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer after teaching, got %v", s.State())
	}

	// The learner can now answer the re-asked question correctly (the small
	// practice bank cycles back to 7 + 3 after 4 + 6), and the loop continues.
	if _, err := tut.SubmitAnswer("Amaka", "10"); err != nil {
		t.Fatalf("expected the correct answer to be accepted, got error: %v", err)
	}
}

// TestServerStopEndsAttemptAndAllowsFreshBegin proves the exit affordance: a
// learner who says "stop" abandons the current attempt (no silent infinite
// loop), and beginning the same topic again starts diagnosis fresh.
func TestServerStopEndsAttemptAndAllowsFreshBegin(t *testing.T) {
	store := session.NewMemoryStore()
	tut := startPracticing(t, store, "Amaka")

	outcome, err := tut.ApplyInput("Amaka", "stop")
	if err != nil {
		t.Fatalf("expected stop to be handled, got error: %v", err)
	}
	if !strings.Contains(outcome.Prompt, "stop here") {
		t.Fatalf("expected a clear stopping message, got %q", outcome.Prompt)
	}

	s, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the session to load, got error: %v", loadErr)
	}
	if s.State() != session.Idle {
		t.Fatalf("expected the session to be idle after stopping, got %v", s.State())
	}

	// Beginning the same topic starts fresh (diagnosis is re-opened).
	if _, err := tut.BeginTopic("Amaka", "Mathematics", "Addition"); err != nil {
		t.Fatalf("expected begin to succeed after stopping, got error: %v", err)
	}
	if _, err := tut.SubmitDiagnosis("Amaka", "I can add single digits"); err != nil {
		t.Fatalf("expected diagnosis to be accepted after stopping, got error: %v", err)
	}
}

// TestServerRecognizesStopSynonyms confirms the common exit words are all
// treated as stop, never as a math answer.
func TestServerRecognizesStopSynonyms(t *testing.T) {
	for _, word := range []string{"quit", "exit", "enough"} {
		store := session.NewMemoryStore()
		tut := startPracticing(t, store, "Amaka")

		outcome, err := tut.ApplyInput("Amaka", word)
		if err != nil {
			t.Fatalf("expected %q to be handled as stop, got error: %v", word, err)
		}
		if !strings.Contains(outcome.Prompt, "stop here") {
			t.Fatalf("expected %q to produce a stopping message, got %q", word, outcome.Prompt)
		}
	}
}
