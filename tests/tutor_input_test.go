package tests

import (
	"errors"
	"testing"

	"soteach/session"
	"soteach/tutor"
)

// TestApplyInputRunsFullDiagnosePracticeVerifyLoop proves the uniform input
// entry (Agent.md §19: tutoring routing stays inside the engine): from only a
// learner's words, the server decides each input is an explanation or an
// answer and drives the entire diagnose -> practice -> verify -> mastery loop
// (README §2) without any client sequencing it.
func TestApplyInputRunsFullDiagnosePracticeVerifyLoop(t *testing.T) {
	store := session.NewMemoryStore()
	tut := tutor.NewTutor(store)

	if _, err := tut.BeginTopic("Amaka", "Mathematics", "Addition", "JSS1-3"); err != nil {
		t.Fatalf("expected begin to succeed, got error: %v", err)
	}

	diag, err := tut.ApplyInput("Amaka", "I can add single digits")
	if err != nil {
		t.Fatalf("expected the diagnostic explanation to be accepted, got error: %v", err)
	}
	if diag.Prompt != "What is 7 + 3?" {
		t.Fatalf("expected the first practice question after diagnosis, got %q", diag.Prompt)
	}

	ans, err := tut.ApplyInput("Amaka", "10")
	if err != nil {
		t.Fatalf("expected the answer to be accepted, got error: %v", err)
	}
	if ans.Prompt != "What is 5 + 6?" {
		t.Fatalf("expected a transfer question after the correct answer, got %q", ans.Prompt)
	}

	verify, err := tut.ApplyInput("Amaka", "11")
	if err != nil {
		t.Fatalf("expected the transfer answer to be accepted, got error: %v", err)
	}
	if verify.Prompt != "That's right! You have mastered Addition." {
		t.Fatalf("expected the completion message, got %q", verify.Prompt)
	}

	s, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the session to load, got error: %v", loadErr)
	}
	if !s.IsMastered() {
		t.Fatal("expected mastery after the full loop")
	}
	if s.State() != session.Mastered {
		t.Fatalf("expected state Mastered, got %v", s.State())
	}
}

// TestApplyInputRoutesIncorrectAnswerBackToPractice confirms the dispatcher
// also routes the wrong-answer case: an incorrect answer is fed back into a
// fresh practice question for the same gap.
func TestApplyInputRoutesIncorrectAnswerBackToPractice(t *testing.T) {
	store := session.NewMemoryStore()
	tut := tutor.NewTutor(store)

	if _, err := tut.BeginTopic("Amaka", "Mathematics", "Addition", "JSS1-3"); err != nil {
		t.Fatalf("expected begin to succeed, got error: %v", err)
	}
	if _, err := tut.ApplyInput("Amaka", "I can add single digits"); err != nil {
		t.Fatalf("expected the diagnostic explanation to be accepted, got error: %v", err)
	}

	outcome, err := tut.ApplyInput("Amaka", "9")
	if err != nil {
		t.Fatalf("expected an incorrect answer to be accepted, got error: %v", err)
	}
	if outcome.Prompt != "What is 4 + 6?" {
		t.Fatalf("expected a fresh question after the wrong answer, got %q", outcome.Prompt)
	}

	s, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the session to load, got error: %v", loadErr)
	}
	if s.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer after the wrong answer, got %v", s.State())
	}
}

// TestApplyInputRejectsWhenNothingAwaiting confirms input is only meaningful
// at a point where the session is actually waiting for it (README §3): after
// mastery there is nothing left to answer in this loop, so further input is
// rejected rather than silently ignored or misrouted.
func TestApplyInputRejectsWhenNothingAwaiting(t *testing.T) {
	store := session.NewMemoryStore()
	tut := tutor.NewTutor(store)

	if _, err := tut.BeginTopic("Amaka", "Mathematics", "Addition", "JSS1-3"); err != nil {
		t.Fatalf("expected begin to succeed, got error: %v", err)
	}
	if _, err := tut.ApplyInput("Amaka", "I can add single digits"); err != nil {
		t.Fatalf("expected the diagnostic explanation to be accepted, got error: %v", err)
	}
	if _, err := tut.ApplyInput("Amaka", "10"); err != nil {
		t.Fatalf("expected the answer to be accepted, got error: %v", err)
	}
	if _, err := tut.ApplyInput("Amaka", "11"); err != nil {
		t.Fatalf("expected the transfer answer to be accepted, got error: %v", err)
	}

	_, err := tut.ApplyInput("Amaka", "more input")
	if err == nil {
		t.Fatal("expected input to be rejected once nothing is awaiting it")
	}
	if !errors.Is(err, tutor.ErrNothingAwaitingInput) {
		t.Fatalf("expected ErrNothingAwaitingInput, got %v", err)
	}

	s, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the session to load, got error: %v", loadErr)
	}
	if s.State() != session.Mastered {
		t.Fatalf("a rejected input must leave the mastered session unchanged, got %v", s.State())
	}
}

// TestApplyInputRequiresABegunSession confirms ApplyInput cannot invent a
// session: a learner with no begun topic has nothing to respond to and must
// BeginTopic first.
func TestApplyInputRequiresABegunSession(t *testing.T) {
	store := session.NewMemoryStore()
	tut := tutor.NewTutor(store)

	_, err := tut.ApplyInput("Amaka", "hello")
	if err == nil {
		t.Fatal("expected input to be rejected for a learner with no begun session")
	}
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}
