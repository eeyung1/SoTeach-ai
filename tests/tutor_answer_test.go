package tests

import (
	"strings"
	"testing"

	"soteach/session"
	"soteach/tutor"
)

// seedDiagnosingSession saves a fresh Addition session for the named learner
// that is mid-diagnosis (subject/topic selected, diagnosis started, not yet
// answered) so a test can drive it through the tutor actions.
func seedDiagnosingSession(t *testing.T, store *session.MemoryStore, learner string) {
	t.Helper()
	s := session.New(learner)
	s.SelectSubject("Mathematics")
	if err := s.SelectTopic("Addition"); err != nil {
		t.Fatalf("expected topic to be selected, got error: %v", err)
	}
	s.StartDiagnosis("What do you already know about addition?")
	store.Save(s)
}

// TestServerRoutesCorrectAnswerToDistinctTransferQuestion is the next beat of
// the loop after TestSubmitDiagnosisAsksFirstPracticeQuestion: a correct
// answer to the first practice question must make the server ask a transfer
// question that is genuinely different (README §2 Stage 4; README §3 rule
// "Correct ≠ understanding") — not repeat the same question and not record
// mastery yet. The client only sent "10"; the server chose what to ask next.
func TestServerRoutesCorrectAnswerToDistinctTransferQuestion(t *testing.T) {
	store := session.NewMemoryStore()
	seedDiagnosingSession(t, store, "Amaka")

	tut := tutor.NewTutor(store)
	if _, err := tut.SubmitDiagnosis("Amaka", "I can add single digits"); err != nil {
		t.Fatalf("expected diagnosis to be accepted, got error: %v", err)
	}

	outcome, err := tut.SubmitAnswer("Amaka", "10")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	const transferQuestion = "What is 5 + 6?"
	if outcome.Prompt != transferQuestion {
		t.Fatalf("expected the server to ask transfer question %q, got %q", transferQuestion, outcome.Prompt)
	}
	if outcome.Prompt == "What is 7 + 3?" {
		t.Fatal("the transfer question must not repeat the original question")
	}

	resumed, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the session to be saved, got error: %v", loadErr)
	}
	if resumed.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer with the transfer question pending, got %v", resumed.State())
	}
	if resumed.IsMastered() {
		t.Fatal("mastery must not be recorded on the first correct answer alone")
	}
}

// TestServerMarksMasteryAfterCorrectTransferAnswer closes the loop: a correct
// answer to the transfer question is genuine evidence of understanding
// (README §3 rule "Correct ≠ understanding") and records mastery.
func TestServerMarksMasteryAfterCorrectTransferAnswer(t *testing.T) {
	store := session.NewMemoryStore()
	seedDiagnosingSession(t, store, "Amaka")

	tut := tutor.NewTutor(store)
	if _, err := tut.SubmitDiagnosis("Amaka", "I can add single digits"); err != nil {
		t.Fatalf("expected diagnosis to be accepted, got error: %v", err)
	}
	if _, err := tut.SubmitAnswer("Amaka", "10"); err != nil {
		t.Fatalf("expected the first answer to be accepted, got error: %v", err)
	}

	const completion = "That's right! You have mastered Addition."
	outcome, err := tut.SubmitAnswer("Amaka", "11")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if outcome.Prompt != completion {
		t.Fatalf("expected completion message %q, got %q", completion, outcome.Prompt)
	}

	resumed, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the session to be saved, got error: %v", loadErr)
	}
	if !resumed.IsMastered() {
		t.Fatal("expected mastery to be recorded after the correct transfer answer")
	}
	if resumed.State() != session.Mastered {
		t.Fatalf("expected state Mastered, got %v", resumed.State())
	}
}

// TestServerRoutesIncorrectAnswerToDifferentQuestion covers the wrong-but-
// attempted branch (README §6, HANDLE_INCORRECT_OR_UNCERTAIN_ANSWER loops
// back to ASK_QUESTION for the same gap): an incorrect answer must not record
// mastery and must be followed by a fresh question, not the identical one.
func TestServerRoutesIncorrectAnswerToDifferentQuestion(t *testing.T) {
	store := session.NewMemoryStore()
	seedDiagnosingSession(t, store, "Amaka")

	tut := tutor.NewTutor(store)
	if _, err := tut.SubmitDiagnosis("Amaka", "I can add single digits"); err != nil {
		t.Fatalf("expected diagnosis to be accepted, got error: %v", err)
	}

	outcome, err := tut.SubmitAnswer("Amaka", "9")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	const retryQuestion = "What is 4 + 6?"
	if outcome.Prompt != retryQuestion {
		t.Fatalf("expected the server to ask retry question %q, got %q", retryQuestion, outcome.Prompt)
	}
	if outcome.Prompt == "What is 7 + 3?" {
		t.Fatal("an incorrect answer must not be followed by the identical question")
	}

	resumed, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the session to be saved, got error: %v", loadErr)
	}
	if resumed.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer with the retry question pending, got %v", resumed.State())
	}
	if resumed.HasAnswer() {
		t.Fatal("the wrong answer must not carry over into the retry")
	}
}

// TestServerTeachesThenReasksAfterUncertainAnswer covers the uncertain branch
// (README §9, explicit uncertainty behavior): an "I don't know" must route to
// teaching first — the server supplies an explanation (Teach) and only then
// asks again (README §8.2's re-teach gate is enforced by the domain).
func TestServerTeachesThenReasksAfterUncertainAnswer(t *testing.T) {
	store := session.NewMemoryStore()
	seedDiagnosingSession(t, store, "Amaka")

	tut := tutor.NewTutor(store)
	if _, err := tut.SubmitDiagnosis("Amaka", "I can add single digits"); err != nil {
		t.Fatalf("expected diagnosis to be accepted, got error: %v", err)
	}

	outcome, err := tut.SubmitAnswer("Amaka", "I don't know")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	const explanation = "Addition means putting two groups together and counting all the items to find the total."
	if !strings.Contains(outcome.Prompt, explanation) {
		t.Fatalf("expected the tutor to teach the gap first, got %q", outcome.Prompt)
	}
	if !strings.Contains(outcome.Prompt, "What is 4 + 6?") {
		t.Fatalf("expected a fresh question after teaching, got %q", outcome.Prompt)
	}

	resumed, loadErr := store.Load("Amaka")
	if loadErr != nil {
		t.Fatalf("expected the session to be saved, got error: %v", loadErr)
	}
	if resumed.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer after teaching and re-asking, got %v", resumed.State())
	}
	if resumed.LastTeaching() != explanation {
		t.Fatalf("expected the explanation to be recorded for auditability, got %q", resumed.LastTeaching())
	}
}

// TestSubmitAnswerRejectedWhenNoQuestionPending confirms the server refuses
// to accept an answer when no question is awaiting one (README §3, "Wait for
// the learner") — e.g. mid-diagnosis there is nothing to answer yet.
func TestSubmitAnswerRejectedWhenNoQuestionPending(t *testing.T) {
	store := session.NewMemoryStore()
	seedDiagnosingSession(t, store, "Amaka")

	tut := tutor.NewTutor(store)
	_, err := tut.SubmitAnswer("Amaka", "10")
	if err == nil {
		t.Fatal("expected SubmitAnswer to be rejected while no question is pending")
	}
}
