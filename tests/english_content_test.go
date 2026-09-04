package tests

import (
	"strings"
	"testing"

	"soteach/session"
	"soteach/tutor"
)

// TestEnglishTopicRunsFullLoopToMastery proves the second subject really
// teaches: English Language / Parts of Speech drives the whole diagnose ->
// practice -> verify -> mastery loop (README §2) with deterministic exact-word
// answers (session.CheckAnswer is exact-match), all in the JSS1-3 voice.
func TestEnglishTopicRunsFullLoopToMastery(t *testing.T) {
	store := session.NewMemoryStore()
	tut := tutor.NewTutor(store)

	begin, err := tut.BeginTopic("Amaka", "English Language", "Parts of Speech", "JSS1-3")
	if err != nil {
		t.Fatalf("expected begin to succeed, got error: %v", err)
	}
	if !strings.Contains(begin.Prompt, "parts of speech, such as") {
		t.Fatalf("expected the English diagnostic prompt, got %q", begin.Prompt)
	}

	const firstQ = "Type the verb in this sentence: 'The diligent student completed her assignment.'"
	diag, err := tut.ApplyInput("Amaka", "I know nouns and verbs")
	if err != nil {
		t.Fatalf("expected the diagnostic explanation to be accepted, got error: %v", err)
	}
	if diag.Prompt != firstQ {
		t.Fatalf("expected the first practice question %q, got %q", firstQ, diag.Prompt)
	}

	const transferQ = "Type the adverb in this sentence: 'The dog barked loudly at night.'"
	transfer, err := tut.ApplyInput("Amaka", "completed")
	if err != nil {
		t.Fatalf("expected the correct answer to be accepted, got error: %v", err)
	}
	if transfer.Prompt != transferQ {
		t.Fatalf("expected the transfer question %q, got %q", transferQ, transfer.Prompt)
	}

	const completion = "That's right! You have mastered the parts of speech."
	verify, err := tut.ApplyInput("Amaka", "loudly")
	if err != nil {
		t.Fatalf("expected the transfer answer to be accepted, got error: %v", err)
	}
	if verify.Prompt != completion {
		t.Fatalf("expected the completion message %q, got %q", completion, verify.Prompt)
	}

	s, err := store.Load("Amaka")
	if err != nil {
		t.Fatalf("expected the session to load, got error: %v", err)
	}
	if !s.IsMastered() {
		t.Fatal("expected mastery after the full English loop")
	}
	if s.Subject() != "English Language" {
		t.Fatalf("expected the session subject to be English Language, got %q", s.Subject())
	}
}

// TestEnglishDiagnosticPromptIsCalibratedByBand proves the second subject's
// opening prompt is age-calibrated like Mathematics (README §4): Primary asks
// about "naming words", JSS about the parts of speech, SSS about function in
// context.
func TestEnglishDiagnosticPromptIsCalibratedByBand(t *testing.T) {
	cases := []struct {
		band   string
		marker string
	}{
		{"Primary 4-6", "naming word"},
		{"JSS1-3", "such as nouns, verbs, and adjectives"},
		{"SSS1-3", "function in a sentence"},
	}
	for _, tc := range cases {
		store := session.NewMemoryStore()
		tut := tutor.NewTutor(store)

		outcome, err := tut.BeginTopic("Amaka", "English Language", "Parts of Speech", tc.band)
		if err != nil {
			t.Fatalf("band %s: expected begin to succeed, got error: %v", tc.band, err)
		}
		if !strings.Contains(outcome.Prompt, tc.marker) {
			t.Fatalf("band %s: expected the diagnostic prompt to contain %q, got %q", tc.band, tc.marker, outcome.Prompt)
		}
	}
}

// TestEnglishWrongAndUncertainAnswersBehaveLikeMath proves the second subject
// follows the same loop semantics as Mathematics: a wrong word does not master,
// an uncertain answer is taught before being re-asked (README §9).
func TestEnglishWrongAndUncertainAnswersBehaveLikeMath(t *testing.T) {
	// Wrong exactness: identifying the wrong word must route back to a fresh
	// question, not to mastery.
	store := session.NewMemoryStore()
	tut := tutor.NewTutor(store)
	if _, err := tut.BeginTopic("Amaka", "English Language", "Parts of Speech", "JSS1-3"); err != nil {
		t.Fatalf("expected begin to succeed, got error: %v", err)
	}
	if _, err := tut.ApplyInput("Amaka", "I know nouns and verbs"); err != nil {
		t.Fatalf("expected the diagnostic explanation to be accepted, got error: %v", err)
	}
	outcome, err := tut.ApplyInput("Amaka", "diligent") // the verb is "completed"
	if err != nil {
		t.Fatalf("expected the wrong answer to be accepted, got error: %v", err)
	}
	const nounQ = "Type the noun in this sentence: 'She reads a book quickly.'"
	if outcome.Prompt != nounQ {
		t.Fatalf("expected a fresh practice question after a wrong word, got %q", outcome.Prompt)
	}

	// Uncertain: taught, then re-asked.
	store2 := session.NewMemoryStore()
	tut2 := tutor.NewTutor(store2)
	if _, err := tut2.BeginTopic("Bello", "English Language", "Parts of Speech", "JSS1-3"); err != nil {
		t.Fatalf("expected begin to succeed, got error: %v", err)
	}
	if _, err := tut2.ApplyInput("Bello", "I know nouns and verbs"); err != nil {
		t.Fatalf("expected the diagnostic explanation to be accepted, got error: %v", err)
	}
	uncertain, err := tut2.ApplyInput("Bello", "I don't know")
	if err != nil {
		t.Fatalf("expected the uncertain answer to be accepted, got error: %v", err)
	}
	if !strings.Contains(uncertain.Prompt, "giving its part of speech") || !strings.Contains(uncertain.Prompt, nounQ) {
		t.Fatalf("expected an uncertain answer to teach the gap then re-ask, got %q", uncertain.Prompt)
	}
}

// TestSubjectSwitchBetweenMathAndEnglishKeepsMasterySeparate proves switching
// subject (Mathematics -> English Language -> Mathematics) resets diagnosis for
// the new subject while preserving per-subject mastery (README §6 mastery is
// tracked per topic within a subject; session.masteryKey is subject|topic).
func TestSubjectSwitchBetweenMathAndEnglishKeepsMasterySeparate(t *testing.T) {
	store := session.NewMemoryStore()
	tut := tutor.NewTutor(store)

	// Master Mathematics / Addition first.
	if _, err := tut.BeginTopic("Amaka", "Mathematics", "Addition", "JSS1-3"); err != nil {
		t.Fatalf("expected begin to succeed, got error: %v", err)
	}
	if _, err := tut.ApplyInput("Amaka", "I can add"); err != nil {
		t.Fatalf("expected the diagnostic explanation to be accepted, got error: %v", err)
	}
	if _, err := tut.ApplyInput("Amaka", "10"); err != nil { // 7 + 3
		t.Fatalf("expected the answer to be accepted, got error: %v", err)
	}
	if _, err := tut.ApplyInput("Amaka", "11"); err != nil { // transfer 5 + 6
		t.Fatalf("expected the transfer answer to be accepted, got error: %v", err)
	}

	// Switch to English: it must begin fresh (a diagnostic prompt), not report
	// the Mathematics mastery.
	outcome, err := tut.BeginTopic("Amaka", "English Language", "Parts of Speech", "JSS1-3")
	if err != nil {
		t.Fatalf("expected the English begin to succeed, got error: %v", err)
	}
	if strings.Contains(outcome.Prompt, "already mastered") {
		t.Fatalf("switching subjects must start fresh diagnosis, got %q", outcome.Prompt)
	}
	if !strings.Contains(outcome.Prompt, "parts of speech, such as") {
		t.Fatalf("expected the English diagnostic prompt, got %q", outcome.Prompt)
	}

	// Switch back to Mathematics / Addition: the earlier mastery must be intact.
	back, err := tut.BeginTopic("Amaka", "Mathematics", "Addition", "JSS1-3")
	if err != nil {
		t.Fatalf("expected the Mathematics resume to succeed, got error: %v", err)
	}
	if back.Prompt != "You have already mastered Addition." {
		t.Fatalf("expected the preserved Addition mastery to be reported, got %q", back.Prompt)
	}
	s, err := store.Load("Amaka")
	if err != nil {
		t.Fatalf("expected the session to load, got error: %v", err)
	}
	if !s.IsMastered() {
		t.Fatal("expected Addition to still be mastered after the subject round-trip")
	}
}
