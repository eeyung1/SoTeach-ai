package tests

import (
	"strings"
	"testing"

	"soteach/session"
	"soteach/tutor"
)

// TestEveryCurriculumTopicHasContentForEveryBand proves the public catalog and
// the content registry never drift apart (workingReadme §3: the curriculum is
// server-owned): every subject/topic/band a client sees must actually begin,
// and each topic's three bands must produce distinct, non-empty diagnostic
// prompts (README §4 calibration). It also locks the topic-name-uniqueness
// invariant that lets content stay subject-agnostic.
func TestEveryCurriculumTopicHasContentForEveryBand(t *testing.T) {
	seen := map[string]bool{}
	for _, subject := range tutor.Curriculum() {
		for _, topic := range subject.Topics {
			if seen[topic] {
				t.Fatalf("topic %q appears under more than one subject; content lookup is by topic name", topic)
			}
			seen[topic] = true

			bandPrompts := map[string]string{}
			for _, band := range session.GradeBands() {
				store := session.NewMemoryStore()
				tut := tutor.NewTutor(store)

				outcome, err := tut.BeginTopic("Amaka", subject.Name, topic, band)
				if err != nil {
					t.Fatalf("subject %s topic %s band %s: expected begin to succeed, got error: %v", subject.Name, topic, band, err)
				}
				if strings.TrimSpace(outcome.Prompt) == "" {
					t.Fatalf("subject %s topic %s band %s: expected a non-empty diagnostic prompt", subject.Name, topic, band)
				}
				bandPrompts[band] = outcome.Prompt
			}

			if len(bandPrompts) != 3 {
				t.Fatalf("subject %s topic %s: expected three band prompts, got %d", subject.Name, topic, len(bandPrompts))
			}
			if bandPrompts["Primary 4-6"] == bandPrompts["JSS1-3"] || bandPrompts["JSS1-3"] == bandPrompts["SSS1-3"] {
				t.Fatalf("subject %s topic %s: grade bands must get distinct diagnostic prompts", subject.Name, topic)
			}
		}
	}
	if !seen["Addition"] || !seen["Parts of Speech"] {
		t.Fatalf("expected the catalog to include Addition and Parts of Speech, got %v", seen)
	}
}

// TestNewMathTopicsDriveFullLoopToMastery proves each new Mathematics topic
// runs the whole diagnose -> practice -> verify -> mastery loop (README §2) in
// the JSS1-3 band with deterministic exact answers, mirroring Addition's loop.
func TestNewMathTopicsDriveFullLoopToMastery(t *testing.T) {
	cases := []struct {
		topic       string
		firstQ      string
		firstAnswer string
		transferQ   string
		transferAns string
		completion  string
	}{
		{
			topic:       "Subtraction",
			firstQ:      "What is 9 - 4?",
			firstAnswer: "5",
			transferQ:   "What is 23 - 15?",
			transferAns: "8",
			completion:  "That's right! You have mastered Subtraction.",
		},
		{
			topic:       "Multiplication",
			firstQ:      "What is 6 × 7?",
			firstAnswer: "42",
			transferQ:   "What is 13 × 5?",
			transferAns: "65",
			completion:  "That's right! You have mastered Multiplication.",
		},
		{
			topic:       "Division",
			firstQ:      "What is 24 ÷ 6?",
			firstAnswer: "4",
			transferQ:   "What is 72 ÷ 9?",
			transferAns: "8",
			completion:  "That's right! You have mastered Division.",
		},
	}
	for _, tc := range cases {
		store := session.NewMemoryStore()
		tut := tutor.NewTutor(store)

		if _, err := tut.BeginTopic("Amaka", "Mathematics", tc.topic, "JSS1-3"); err != nil {
			t.Fatalf("topic %s: expected begin to succeed, got error: %v", tc.topic, err)
		}

		diag, err := tut.ApplyInput("Amaka", "I can do "+tc.topic+" in my head")
		if err != nil {
			t.Fatalf("topic %s: expected the diagnostic explanation to be accepted, got error: %v", tc.topic, err)
		}
		if diag.Prompt != tc.firstQ {
			t.Fatalf("topic %s: expected the first practice question %q, got %q", tc.topic, tc.firstQ, diag.Prompt)
		}

		transfer, err := tut.ApplyInput("Amaka", tc.firstAnswer)
		if err != nil {
			t.Fatalf("topic %s: expected the answer to be accepted, got error: %v", tc.topic, err)
		}
		if transfer.Prompt != tc.transferQ {
			t.Fatalf("topic %s: expected the transfer question %q, got %q", tc.topic, tc.transferQ, transfer.Prompt)
		}

		verify, err := tut.ApplyInput("Amaka", tc.transferAns)
		if err != nil {
			t.Fatalf("topic %s: expected the transfer answer to be accepted, got error: %v", tc.topic, err)
		}
		if verify.Prompt != tc.completion {
			t.Fatalf("topic %s: expected the completion message %q, got %q", tc.topic, tc.completion, verify.Prompt)
		}

		s, err := store.Load("Amaka")
		if err != nil {
			t.Fatalf("topic %s: expected the session to load, got error: %v", tc.topic, err)
		}
		if !s.IsMastered() {
			t.Fatalf("topic %s: expected mastery after the full loop", tc.topic)
		}
	}
}

// TestNewMathTopicsAreBandCalibrated proves each new Mathematics topic speaks
// in the three distinct band voices (README §4 / Agent.md §13), not just
// Addition — the same calibration requirement the age-band tests enforce for
// Addition.
func TestNewMathTopicsAreBandCalibrated(t *testing.T) {
	cases := []struct {
		topic   string
		primary string
		jss     string
		sss     string
	}{
		{topic: "Subtraction", primary: "taking away", jss: "know about subtraction", sss: "minuend, subtrahend"},
		{topic: "Multiplication", primary: "count things in groups", jss: "about multiplication", sss: "commutativity and identity"},
		{topic: "Division", primary: "share sweets fairly", jss: "about division", sss: "dividend, divisor, and quotient"},
	}
	for _, tc := range cases {
		expect := map[string]string{
			"Primary 4-6": tc.primary,
			"JSS1-3":      tc.jss,
			"SSS1-3":      tc.sss,
		}
		for band, marker := range expect {
			store := session.NewMemoryStore()
			tut := tutor.NewTutor(store)

			outcome, err := tut.BeginTopic("Amaka", "Mathematics", tc.topic, band)
			if err != nil {
				t.Fatalf("topic %s band %s: expected begin to succeed, got error: %v", tc.topic, band, err)
			}
			if !strings.Contains(outcome.Prompt, marker) {
				t.Fatalf("topic %s band %s: expected the diagnostic prompt to contain %q, got %q", tc.topic, band, marker, outcome.Prompt)
			}
		}
	}
}

// TestNewMathTopicWrongAndUncertainBehaveLikeAddition proves the loop semantics
// for a new topic match Addition's: a wrong answer loops back to a fresh
// practice question, and an uncertain answer is taught before being re-asked
// (README §9), never falsely marking mastery.
func TestNewMathTopicWrongAndUncertainBehaveLikeAddition(t *testing.T) {
	// Wrong answer: after diagnosing Subtraction in JSS1-3, an incorrect
	// answer must route to a fresh practice question for the same gap.
	store := session.NewMemoryStore()
	tut := tutor.NewTutor(store)
	if _, err := tut.BeginTopic("Amaka", "Mathematics", "Subtraction", "JSS1-3"); err != nil {
		t.Fatalf("expected begin to succeed, got error: %v", err)
	}
	if _, err := tut.ApplyInput("Amaka", "I can subtract"); err != nil {
		t.Fatalf("expected the diagnostic explanation to be accepted, got error: %v", err)
	}
	outcome, err := tut.ApplyInput("Amaka", "3") // 9 - 4 is 5
	if err != nil {
		t.Fatalf("expected the wrong answer to be accepted, got error: %v", err)
	}
	if outcome.Prompt != "What is 15 - 6?" {
		t.Fatalf("expected a fresh practice question after a wrong answer, got %q", outcome.Prompt)
	}

	// Uncertain answer: the learner is taught the explanation, then re-asked.
	store2 := session.NewMemoryStore()
	tut2 := tutor.NewTutor(store2)
	if _, err := tut2.BeginTopic("Bello", "Mathematics", "Subtraction", "JSS1-3"); err != nil {
		t.Fatalf("expected begin to succeed, got error: %v", err)
	}
	if _, err := tut2.ApplyInput("Bello", "I can subtract"); err != nil {
		t.Fatalf("expected the diagnostic explanation to be accepted, got error: %v", err)
	}
	uncertain, err := tut2.ApplyInput("Bello", "I don't know")
	if err != nil {
		t.Fatalf("expected the uncertain answer to be accepted, got error: %v", err)
	}
	if !strings.Contains(uncertain.Prompt, "minuend") || !strings.Contains(uncertain.Prompt, "What is 15 - 6?") {
		t.Fatalf("expected an uncertain answer to teach the gap then re-ask, got %q", uncertain.Prompt)
	}
}
