package tests

import (
	"errors"
	"strings"
	"testing"

	"soteach/session"
	"soteach/tutor"
)

// TestBeginTopicDiagnosticPromptIsCalibratedByBand proves README §4/Agent.md
// §13 non-cosmetic calibration: the very first prompt a learner sees already
// reflects their grade band's voice.
func TestBeginTopicDiagnosticPromptIsCalibratedByBand(t *testing.T) {
	cases := []struct {
		band   string
		marker string
	}{
		{"Primary 4-6", "2 + 2"},
		{"JSS1-3", "What do you already know about addition?"},
		{"SSS1-3", "commutativity"},
	}
	for _, tc := range cases {
		store := session.NewMemoryStore()
		tut := tutor.NewTutor(store)

		outcome, err := tut.BeginTopic("Amaka", "Mathematics", "Addition", tc.band)
		if err != nil {
			t.Fatalf("band %s: expected begin to succeed, got error: %v", tc.band, err)
		}
		if !strings.Contains(outcome.Prompt, tc.marker) {
			t.Fatalf("band %s: expected the diagnostic prompt to contain %q, got %q", tc.band, tc.marker, outcome.Prompt)
		}
	}
}

// TestFirstPracticeQuestionReflectsBand proves the practice voice differs too:
// Primary gets a concrete word problem, JSS the plain numeric question, SSS an
// exam-style one — and each is still deterministically answerable (a correct
// answer advances to a band-calibrated transfer question).
func TestFirstPracticeQuestionReflectsBand(t *testing.T) {
	cases := []struct {
		band        string
		questionHas string
		transferHas string
	}{
		{"Primary 4-6", "sweets", "buns"},
		{"JSS1-3", "What is 7 + 3?", "What is 5 + 6?"},
		{"SSS1-3", "Evaluate: 7 + 3.", "Determine the value of 5 + 6."},
	}
	for _, tc := range cases {
		store := session.NewMemoryStore()
		tut := tutor.NewTutor(store)

		if _, err := tut.BeginTopic("Amaka", "Mathematics", "Addition", tc.band); err != nil {
			t.Fatalf("band %s: expected begin to succeed, got error: %v", tc.band, err)
		}
		practice, err := tut.SubmitDiagnosis("Amaka", "I add single digits but get stuck when it carries over")
		if err != nil {
			t.Fatalf("band %s: expected diagnosis to be accepted, got error: %v", tc.band, err)
		}
		if !strings.Contains(practice.Prompt, tc.questionHas) {
			t.Fatalf("band %s: expected the first question to contain %q, got %q", tc.band, tc.questionHas, practice.Prompt)
		}

		transfer, err := tut.SubmitAnswer("Amaka", "10")
		if err != nil {
			t.Fatalf("band %s: expected the correct answer to be accepted, got error: %v", tc.band, err)
		}
		if !strings.Contains(transfer.Prompt, tc.transferHas) {
			t.Fatalf("band %s: expected the transfer question to contain %q, got %q", tc.band, tc.transferHas, transfer.Prompt)
		}
	}
}

// TestExplanationCalibratedByBand proves the teaching voice differs when a
// learner is uncertain ("I don't know"), per band.
func TestExplanationCalibratedByBand(t *testing.T) {
	cases := []struct {
		band   string
		marker string
	}{
		{"Primary 4-6", "sweets"},
		{"JSS1-3", "groups"},
		{"SSS1-3", "addends"},
	}
	for _, tc := range cases {
		store := session.NewMemoryStore()
		tut := tutor.NewTutor(store)

		if _, err := tut.BeginTopic("Amaka", "Mathematics", "Addition", tc.band); err != nil {
			t.Fatalf("band %s: expected begin to succeed, got error: %v", tc.band, err)
		}
		if _, err := tut.SubmitDiagnosis("Amaka", "I can add single digits"); err != nil {
			t.Fatalf("band %s: expected diagnosis to be accepted, got error: %v", tc.band, err)
		}

		outcome, err := tut.ApplyInput("Amaka", "I don't know")
		if err != nil {
			t.Fatalf("band %s: expected the uncertain answer to be handled, got error: %v", tc.band, err)
		}
		if !strings.Contains(outcome.Prompt, tc.marker) {
			t.Fatalf("band %s: expected the explanation to contain %q, got %q", tc.band, tc.marker, outcome.Prompt)
		}
	}
}

// TestBeginTopicRejectsInvalidGradeBand confirms begin refuses an undefined
// band without creating a session (README §4 bands are enforced).
func TestBeginTopicRejectsInvalidGradeBand(t *testing.T) {
	store := session.NewMemoryStore()
	tut := tutor.NewTutor(store)

	_, err := tut.BeginTopic("Amaka", "Mathematics", "Addition", "Grade 9")
	if err == nil {
		t.Fatal("expected an undefined grade band to be rejected")
	}
	if !errors.Is(err, session.ErrInvalidGradeBand) {
		t.Fatalf("expected ErrInvalidGradeBand, got %v", err)
	}

	if _, loadErr := store.Load("Amaka"); loadErr == nil {
		t.Fatal("a rejected begin must not create a session")
	}
}
