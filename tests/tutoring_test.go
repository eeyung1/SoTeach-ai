package tests

import (
	"testing"

	"soteach/session"
)

func TestTutorWaitsForLearnerAnswer(t *testing.T) {
	s := session.New("Amaka")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")

	s.AskQuestion("What is 7 + 3?", "10")

	if s.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer after asking a question, got %v", s.State())
	}

	if s.HasAnswer() {
		t.Fatal("session should not have an answer before the learner responds")
	}
}

func TestSubmitAnswerMovesToCheckAnswerState(t *testing.T) {
	s := session.New("Amaka")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
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
	s := session.New("Amaka")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
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

// TestCorrectMathAnswerIsMarkedCorrect checks that a matching answer is
// recognized as correct. It no longer asserts a terminal "Correct" state
// (README §3, rule "Correct ≠ understanding" — see
// TestCorrectAnswerDoesNotImmediatelyRecordMastery below for the state
// this now transitions into).
func TestCorrectMathAnswerIsMarkedCorrect(t *testing.T) {
	s := session.New("Amaka")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")
	s.SubmitAnswer("10")

	correct := s.CheckAnswer()

	if !correct {
		t.Fatal("10 must be marked correct for 7 + 3")
	}
}

// TestCorrectAnswerDoesNotImmediatelyRecordMastery is a direct test of
// README §3, rule "Correct ≠ understanding": a right answer can come from
// guessing, memorizing, or copying, so a single correct answer must not be
// recorded as mastery. The session must move to a state that still expects
// a transfer/verification question before mastery can be recorded.
func TestCorrectAnswerDoesNotImmediatelyRecordMastery(t *testing.T) {
	s := session.New("Amaka")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")
	s.SubmitAnswer("10")

	correct := s.CheckAnswer()

	if !correct {
		t.Fatal("10 must be marked correct for 7 + 3")
	}

	if s.IsMastered() {
		t.Fatal("a single correct answer must not record mastery")
	}

	if s.State() != session.AwaitingTransferCheck {
		t.Fatalf("expected state AwaitingTransferCheck after a correct answer, got %v", s.State())
	}
}

// TestConfirmationCountDoesNotExceedTwo is a direct test of README §3,
// rule "Confirmation count is exact": exactly two confirmations per
// answer, tracked as explicit state — never three or four, and never left
// to the LLM's own conversational memory.
func TestConfirmationCountDoesNotExceedTwo(t *testing.T) {
	s := session.New("Amaka")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")
	s.SubmitAnswer("10")
	s.CheckAnswer()

	first := s.RequestConfirmation()
	if !first {
		t.Fatal("first confirmation request should be allowed")
	}

	second := s.RequestConfirmation()
	if !second {
		t.Fatal("second confirmation request should be allowed")
	}

	third := s.RequestConfirmation()
	if third {
		t.Fatal("a third confirmation request must not be allowed")
	}

	if s.ConfirmationCount() != 2 {
		t.Fatalf("expected confirmation count to stay at 2, got %d", s.ConfirmationCount())
	}
}

// TestLearnerStateRemainsSeparate is a direct test of README §3, rule
// "Multiple learners, separate state": a session may involve several
// named learners with individual turns and separate progress — one
// learner's state must never be merged or confused with another's.
func TestLearnerStateRemainsSeparate(t *testing.T) {
	amaka := session.New("Amaka")
	bello := session.New("Bello")

	if amaka.LearnerName() != "Amaka" {
		t.Fatalf("expected learner name Amaka, got %s", amaka.LearnerName())
	}
	if bello.LearnerName() != "Bello" {
		t.Fatalf("expected learner name Bello, got %s", bello.LearnerName())
	}

	amaka.StartDiagnosis("What do you already know about addition?")
	amaka.RecordDiagnosticResponse("I know how to add single digits")
	amaka.AskQuestion("What is 7 + 3?", "10")
	amaka.SubmitAnswer("10")
	amaka.CheckAnswer()
	amaka.RequestConfirmation()

	if bello.State() != session.Idle {
		t.Fatalf("Bello's state must remain Idle while Amaka progresses, got %v", bello.State())
	}
	if bello.HasAnswer() {
		t.Fatal("Bello must not have an answer recorded from Amaka's session")
	}
	if bello.ConfirmationCount() != 0 {
		t.Fatalf("Bello's confirmation count must remain 0, got %d", bello.ConfirmationCount())
	}
}

// TestDiagnosisCapturesLearnerExplanationBeforeTeaching is a direct test
// of README §2, Stage 1 (DIAGNOSE): before any teaching happens, the
// session must ask the learner to explain what they already know, and
// record that explanation, so the real gap can be located instead of
// assumed.
func TestDiagnosisCapturesLearnerExplanationBeforeTeaching(t *testing.T) {
	s := session.New("Amaka")

	if s.DiagnosisComplete() {
		t.Fatal("a new session must not start with diagnosis already complete")
	}

	s.StartDiagnosis("What do you already know about fractions?")

	if s.State() != session.Diagnosing {
		t.Fatalf("expected state Diagnosing after StartDiagnosis, got %v", s.State())
	}

	s.RecordDiagnosticResponse("I know how to divide but I don't know what the top number means")

	if !s.DiagnosisComplete() {
		t.Fatal("diagnosis should be complete after recording the learner's response")
	}

	if s.DiagnosticFinding() != "I know how to divide but I don't know what the top number means" {
		t.Fatalf("expected diagnostic finding to be recorded verbatim, got %q", s.DiagnosticFinding())
	}
}

// TestAskQuestionIsRejectedBeforeDiagnosisIsComplete is a direct test of
// README §2: "No stage is optional, and no stage may be silently skipped
// by the model." TEACH/PRACTICE (represented here by AskQuestion) must not
// be reachable until DIAGNOSE has recorded a finding.
func TestAskQuestionIsRejectedBeforeDiagnosisIsComplete(t *testing.T) {
	s := session.New("Amaka")

	err := s.AskQuestion("What is 7 + 3?", "10")

	if err == nil {
		t.Fatal("AskQuestion must be rejected before diagnosis is complete")
	}

	if s.State() != session.Idle {
		t.Fatalf("state must remain Idle when AskQuestion is rejected, got %v", s.State())
	}

	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")

	err = s.AskQuestion("What is 7 + 3?", "10")

	if err != nil {
		t.Fatalf("AskQuestion should be allowed once diagnosis is complete, got error: %v", err)
	}

	if s.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer after AskQuestion post-diagnosis, got %v", s.State())
	}
}

// TestMasteryIsRecordedOnlyAfterCorrectTransferAnswer is a direct test of
// README §3, rule "Correct ≠ understanding": after an initial correct
// answer, the session must pose a transfer question (a different but
// related problem) and only record mastery if that is also answered
// correctly. It must reject a transfer question that is identical to the
// original — repetition is not verification (README §2, Stage 4: VERIFY
// "using a genuinely different question, not a repeat of the same one").
func TestMasteryIsRecordedOnlyAfterCorrectTransferAnswer(t *testing.T) {
	s := session.New("Amaka")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")
	s.SubmitAnswer("10")
	s.CheckAnswer()

	if s.State() != session.AwaitingTransferCheck {
		t.Fatalf("expected state AwaitingTransferCheck before transfer question, got %v", s.State())
	}

	err := s.AskTransferQuestion("What is 7 + 3?", "10")
	if err == nil {
		t.Fatal("a transfer question identical to the original must be rejected")
	}

	err = s.AskTransferQuestion("What is 4 + 6?", "10")
	if err != nil {
		t.Fatalf("a genuinely different transfer question should be accepted, got error: %v", err)
	}

	s.SubmitAnswer("10")
	transferCorrect := s.CheckAnswer()

	if !transferCorrect {
		t.Fatal("10 must be marked correct for 4 + 6")
	}

	if !s.IsMastered() {
		t.Fatal("mastery should be recorded after a correct transfer answer")
	}

	if s.State() != session.Mastered {
		t.Fatalf("expected state Mastered after correct transfer answer, got %v", s.State())
	}
}

// TestIncorrectAnswerLoopsBackToNewQuestionForSameGap is a direct test of
// README §6's state machine: HANDLE_INCORRECT_OR_UNCERTAIN_ANSWER loops
// back to ASK_QUESTION for the same gap. Diagnosis must not need to be
// repeated (README §6: "must support resuming ... without repeating the
// diagnostic stage from scratch"), and stale per-question state (the
// previous wrong answer, confirmation count, transfer-question flag) must
// not leak into the retry.
func TestIncorrectAnswerLoopsBackToNewQuestionForSameGap(t *testing.T) {
	s := session.New("Amaka")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")
	s.SubmitAnswer("9")
	s.CheckAnswer()

	if s.State() != session.Incorrect {
		t.Fatalf("expected state Incorrect after wrong answer, got %v", s.State())
	}

	err := s.AskQuestion("What is 2 + 3?", "5")
	if err != nil {
		t.Fatalf("looping back to a new question after an incorrect answer should be allowed, got error: %v", err)
	}

	if s.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer after looping back, got %v", s.State())
	}

	if s.HasAnswer() {
		t.Fatal("the previous wrong answer must not carry over into the retry")
	}

	if s.ConfirmationCount() != 0 {
		t.Fatalf("confirmation count must reset on retry, got %d", s.ConfirmationCount())
	}

	s.SubmitAnswer("5")
	correct := s.CheckAnswer()

	if !correct {
		t.Fatal("5 must be marked correct for 2 + 3")
	}

	if s.State() != session.AwaitingTransferCheck {
		t.Fatalf("expected state AwaitingTransferCheck after correct retry answer, got %v", s.State())
	}
}

// TestAskQuestionIsRejectedWhileAnswerIsPending is a direct test of
// README §3, rule "Wait for the learner": "Never advance to the next
// question while one is pending." Calling AskQuestion while a question is
// already awaiting an answer must be rejected, and the pending question
// must be left completely unchanged.
func TestAskQuestionIsRejectedWhileAnswerIsPending(t *testing.T) {
	s := session.New("Amaka")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")

	if err := s.AskQuestion("What is 7 + 3?", "10"); err != nil {
		t.Fatalf("first AskQuestion should be allowed, got error: %v", err)
	}

	err := s.AskQuestion("What is 2 + 2?", "4")
	if err == nil {
		t.Fatal("AskQuestion must be rejected while a question is already pending")
	}

	if s.State() != session.WaitForAnswer {
		t.Fatalf("state must remain WaitForAnswer after a rejected AskQuestion, got %v", s.State())
	}

	s.SubmitAnswer("10")
	correct := s.CheckAnswer()

	if !correct {
		t.Fatal("the original pending question (7 + 3 = 10) must still be in effect after a rejected AskQuestion")
	}
}
