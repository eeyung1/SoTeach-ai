package tests

import (
	"testing"

	"soteach/session"
)

func TestTutorWaitsForLearnerAnswer(t *testing.T) {
	s := session.New()

	s.AskQuestion("What is 7 + 3?", "10")

	if s.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer after asking a question, got %v", s.State())
	}

	if s.HasAnswer() {
		t.Fatal("session should not have an answer before the learner responds")
	}
}

func TestSubmitAnswerMovesToCheckAnswerState(t *testing.T) {
	s := session.New()
	s.AskQuestion("What is 7 + 3?", "10")

	s.SubmitAnswer("10")

	if s.State() != session.CheckAnswer {
		t.Fatalf("expected state CheckAnswer after submitting an answer, got %v", s.State())
	}

	if !s.HasAnswer() {
		t.Fatal("session should have an answer recorded after SubmitAnswer")
	}
}

// TestIncorrectMathAnswerIsNotMarkedCorrect is a direct regression test for
// the real observed failure in README §3: "7 + 3 was initially accepted as
// correct when the answer given was wrong."
func TestIncorrectMathAnswerIsNotMarkedCorrect(t *testing.T) {
	s := session.New()
	s.AskQuestion("What is 7 + 3?", "10")
	s.SubmitAnswer("9")

	correct := s.CheckAnswer()

	if correct {
		t.Fatal("9 must not be marked correct for 7 + 3")
	}

	if s.State() != session.Incorrect {
		t.Fatalf("expected state Incorrect, got %v", s.State())
	}
}

func TestCorrectMathAnswerIsMarkedCorrect(t *testing.T) {
	s := session.New()
	s.AskQuestion("What is 7 + 3?", "10")
	s.SubmitAnswer("10")

	correct := s.CheckAnswer()

	if !correct {
		t.Fatal("10 must be marked correct for 7 + 3")
	}

	if s.State() != session.Correct {
		t.Fatalf("expected state Correct, got %v", s.State())
	}
}
