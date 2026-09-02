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

// TestCurrentQuestionCanBeRepeatedWithoutLosingState is a direct test of
// README §3, "Handle interruptions without losing state": a learner
// asking "what was the question?" must be answerable without losing the
// pending question, the answer state, or the confirmation count.
func TestCurrentQuestionCanBeRepeatedWithoutLosingState(t *testing.T) {
	s := session.New("Amaka")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")

	first := s.CurrentQuestion()
	if first != "What is 7 + 3?" {
		t.Fatalf("expected current question %q, got %q", "What is 7 + 3?", first)
	}

	second := s.CurrentQuestion()
	if second != first {
		t.Fatalf("repeating the question changed the result: got %q, want %q", second, first)
	}

	if s.State() != session.WaitForAnswer {
		t.Fatalf("repeating the question must not change state, got %v", s.State())
	}
	if s.HasAnswer() {
		t.Fatal("repeating the question must not create an answer")
	}
	if s.ConfirmationCount() != 0 {
		t.Fatalf("repeating the question must not affect confirmation count, got %d", s.ConfirmationCount())
	}

	s.SubmitAnswer("10")
	if !s.CheckAnswer() {
		t.Fatal("session must still check answers correctly after the question was repeated")
	}
}

// TestRequestConfirmationRejectedBeforeAnswerIsChecked is a direct test
// of README §8.3, "exact confirmation count": confirmation only makes
// sense once an answer has actually been checked. Calling
// RequestConfirmation while idle, mid-diagnosis, or still waiting for the
// learner's answer must be rejected and must not consume the count.
func TestRequestConfirmationRejectedBeforeAnswerIsChecked(t *testing.T) {
	s := session.New("Amaka")

	if s.RequestConfirmation() {
		t.Fatal("confirmation must be rejected before diagnosis has even started")
	}

	s.StartDiagnosis("What do you already know about addition?")
	if s.RequestConfirmation() {
		t.Fatal("confirmation must be rejected while diagnosis is still in progress")
	}

	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")
	if s.RequestConfirmation() {
		t.Fatal("confirmation must be rejected while still waiting for the learner's answer")
	}

	if s.ConfirmationCount() != 0 {
		t.Fatalf("rejected confirmation attempts must not consume the count, got %d", s.ConfirmationCount())
	}

	s.SubmitAnswer("10")
	s.CheckAnswer()

	if !s.RequestConfirmation() {
		t.Fatal("confirmation should be allowed once the answer has been checked")
	}
}

// TestUncertainAnswerIsNotMarkedIncorrectOrCorrect is a direct test of
// README §9, "Explicit uncertainty behavior — the system must not pretend
// to know something it's uncertain about", and Agent.md §14's required
// "uncertain/ambiguous answers" test category. A learner admitting they
// don't know must be distinguished from a genuine wrong guess: not marked
// correct, but also not folded into the same Incorrect state as an actual
// incorrect attempt.
func TestUncertainAnswerIsNotMarkedIncorrectOrCorrect(t *testing.T) {
	s := session.New("Amaka")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")
	s.SubmitAnswer("I don't know")

	correct := s.CheckAnswer()

	if correct {
		t.Fatal("an uncertain answer must not be marked correct")
	}

	if s.State() != session.Uncertain {
		t.Fatalf("expected state Uncertain for an 'I don't know' response, got %v", s.State())
	}

	if s.State() == session.Incorrect {
		t.Fatal("an uncertain answer must be distinguished from a genuine incorrect guess")
	}
}

// TestUncertainAnswerRecognizesCommonPhrasing checks that a couple of
// common variants ("idk", "not sure") are recognized the same way, since
// the behavior being tested is phrase recognition, not a new feature per
// variant (Agent.md §55, test scope discipline).
func TestUncertainAnswerRecognizesCommonPhrasing(t *testing.T) {
	for _, phrase := range []string{"idk", "not sure", "Not Sure"} {
		s := session.New("Amaka")
		s.StartDiagnosis("What do you already know about addition?")
		s.RecordDiagnosticResponse("I know how to add single digits")
		s.AskQuestion("What is 7 + 3?", "10")
		s.SubmitAnswer(phrase)

		if correct := s.CheckAnswer(); correct {
			t.Fatalf("phrase %q must not be marked correct", phrase)
		}

		if s.State() != session.Uncertain {
			t.Fatalf("phrase %q: expected state Uncertain, got %v", phrase, s.State())
		}
	}
}

// TestAskQuestionIsRejectedAfterUncertainAnswerUntilTaught is a direct
// test of the product decision that an uncertain answer ("I don't know")
// must route toward re-teaching, not straight back into another practice
// question. This supersedes the earlier assumption (that Uncertain and
// Incorrect loop back identically) — see
// TestIncorrectAnswerDoesNotRequireTeaching below for the contrast this
// decision draws with a genuine wrong-but-attempted answer.
func TestAskQuestionIsRejectedAfterUncertainAnswerUntilTaught(t *testing.T) {
	s := session.New("Amaka")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")
	s.SubmitAnswer("I don't know")
	s.CheckAnswer()

	if s.State() != session.Uncertain {
		t.Fatalf("expected state Uncertain after 'I don't know', got %v", s.State())
	}

	err := s.AskQuestion("What is 2 + 3?", "5")
	if err == nil {
		t.Fatal("AskQuestion must be rejected after an uncertain answer until the gap has been taught")
	}

	if s.State() != session.Uncertain {
		t.Fatalf("state must remain Uncertain after a rejected AskQuestion, got %v", s.State())
	}
}

// TestUncertainAnswerLoopsBackToNewQuestionAfterTeaching confirms that
// once Teach() has been called for the gap surfaced by an uncertain
// answer, the loop continues normally: AskQuestion is allowed again, and
// prior answer/confirmation state does not carry over into the retry
// (README §6, HANDLE_INCORRECT_OR_UNCERTAIN_ANSWER loops back to
// ASK_QUESTION).
func TestUncertainAnswerLoopsBackToNewQuestionAfterTeaching(t *testing.T) {
	s := session.New("Amaka")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")
	s.SubmitAnswer("I don't know")
	s.CheckAnswer()

	s.Teach("Adding means combining two numbers into a total.")

	err := s.AskQuestion("What is 2 + 3?", "5")
	if err != nil {
		t.Fatalf("looping back to a new question after teaching should be allowed, got error: %v", err)
	}

	if s.State() != session.WaitForAnswer {
		t.Fatalf("expected state WaitForAnswer after looping back, got %v", s.State())
	}

	if s.HasAnswer() {
		t.Fatal("the previous uncertain answer must not carry over into the retry")
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

// TestIncorrectAnswerDoesNotRequireTeaching is a regression guard for the
// product decision just introduced: a wrong-but-attempted answer is NOT
// treated the same as an uncertain one. Incorrect must still be allowed
// to loop straight back to AskQuestion without an intervening Teach()
// call.
func TestIncorrectAnswerDoesNotRequireTeaching(t *testing.T) {
	s := session.New("Amaka")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")
	s.SubmitAnswer("9")
	s.CheckAnswer()

	if s.State() != session.Incorrect {
		t.Fatalf("expected state Incorrect, got %v", s.State())
	}

	err := s.AskQuestion("What is 2 + 3?", "5")
	if err != nil {
		t.Fatalf("Incorrect must not require Teach() before looping back, got error: %v", err)
	}
}

// TestTeachRecordsExplanation is a direct test of README §9 safeguard 6
// (auditability) and §10 (learner state must track diagnostic
// findings/concepts needing work): Teach() must retain the explanation it
// was given, not just flip an internal flag, so it can later be surfaced
// (e.g. to a parent/teacher view or an audit log).
func TestTeachRecordsExplanation(t *testing.T) {
	s := session.New("Amaka")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")
	s.SubmitAnswer("I don't know")
	s.CheckAnswer()

	s.Teach("Adding means combining two numbers into a total.")

	if got := s.LastTeaching(); got != "Adding means combining two numbers into a total." {
		t.Fatalf("expected LastTeaching to return the given explanation, got %q", got)
	}
}

// TestLastTeachingIsEmptyBeforeAnyTeaching confirms there is no stale or
// placeholder value before Teach() has ever been called.
func TestLastTeachingIsEmptyBeforeAnyTeaching(t *testing.T) {
	s := session.New("Amaka")

	if got := s.LastTeaching(); got != "" {
		t.Fatalf("expected empty LastTeaching before any Teach() call, got %q", got)
	}
}

// TestSubmitAnswerFromWrongLearnerIsRejected is a direct test of
// Blueprint lesson #1 ("The AI needs explicit turn management") and
// README §3, rule "One active respondent": "A learner's response must
// never accidentally be attributed to another learner." A multi-learner
// context (session.Roster) must track whose turn it is and reject an
// answer submitted on behalf of a learner who is not currently active,
// without affecting either learner's underlying Session state.
func TestSubmitAnswerFromWrongLearnerIsRejected(t *testing.T) {
	r := session.NewRoster("Amaka", "Bello")

	amaka := r.Session("Amaka")
	amaka.StartDiagnosis("What do you already know about addition?")
	amaka.RecordDiagnosticResponse("I know how to add single digits")
	amaka.AskQuestion("What is 7 + 3?", "10")

	if r.ActiveLearner() != "Amaka" {
		t.Fatalf("expected Amaka to be the active learner, got %s", r.ActiveLearner())
	}

	err := r.SubmitAnswerFor("Bello", "10")
	if err == nil {
		t.Fatal("submitting an answer for a learner who is not active must be rejected")
	}

	if amaka.HasAnswer() {
		t.Fatal("Amaka's session must be unaffected by an answer submitted for Bello")
	}

	bello := r.Session("Bello")
	if bello.HasAnswer() {
		t.Fatal("Bello's session must not receive an answer that was rejected")
	}

	err = r.SubmitAnswerFor("Amaka", "10")
	if err != nil {
		t.Fatalf("submitting an answer for the active learner should be allowed, got error: %v", err)
	}

	if !amaka.HasAnswer() {
		t.Fatal("Amaka's session should have the answer recorded once correctly attributed")
	}
}

// TestSubmitAnswerForUnknownLearnerIsRejected confirms a Roster only
// accepts answers from learners actually in the session, per README §3's
// "one active respondent" rule extending naturally to "one of the
// participating respondents."
func TestSubmitAnswerForUnknownLearnerIsRejected(t *testing.T) {
	r := session.NewRoster("Amaka", "Bello")

	err := r.SubmitAnswerFor("Precious", "10")
	if err == nil {
		t.Fatal("submitting an answer for a learner not in the roster must be rejected")
	}
}

// TestSubjectDefaultsEmpty confirms a new session has no subject
// selected until SelectSubject is called (Blueprint lesson #9: "The
// system must handle subject switching, such as Mathematics to
// Christian Religious Studies").
func TestSubjectDefaultsEmpty(t *testing.T) {
	s := session.New("Amaka")
	if s.Subject() != "" {
		t.Fatalf("expected empty subject before SelectSubject, got %q", s.Subject())
	}
}

// TestSwitchingSubjectResetsDiagnosisState is a direct test of Blueprint
// lesson #9. Switching to a genuinely different subject must reset
// diagnosis (the new subject has not been diagnosed yet) and clear any
// pending question, answer, and confirmation state left over from the
// previous subject.
func TestSwitchingSubjectResetsDiagnosisState(t *testing.T) {
	s := session.New("Amaka")
	s.SelectSubject("Mathematics")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")

	s.SelectSubject("Christian Religious Studies")

	if s.Subject() != "Christian Religious Studies" {
		t.Fatalf("expected subject to update, got %q", s.Subject())
	}

	if s.DiagnosisComplete() {
		t.Fatal("switching subjects must reset diagnosis for the new subject")
	}

	if s.State() != session.Idle {
		t.Fatalf("expected state Idle after switching subjects, got %v", s.State())
	}

	if s.HasAnswer() {
		t.Fatal("switching subjects must not carry over the previous subject's answer")
	}

	if s.CurrentQuestion() != "" {
		t.Fatalf("switching subjects must clear the pending question, got %q", s.CurrentQuestion())
	}

	if s.ConfirmationCount() != 0 {
		t.Fatalf("switching subjects must reset confirmation count, got %d", s.ConfirmationCount())
	}

	err := s.AskQuestion("Who created the world?", "God")
	if err == nil {
		t.Fatal("AskQuestion must be rejected until the new subject has been diagnosed")
	}
}

// TestSelectingSameSubjectDoesNotResetState confirms SelectSubject is
// idempotent for the current subject — re-selecting the subject already
// in progress must not destroy in-progress diagnosis or question state.
func TestSelectingSameSubjectDoesNotResetState(t *testing.T) {
	s := session.New("Amaka")
	s.SelectSubject("Mathematics")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")

	s.SelectSubject("Mathematics")

	if s.State() != session.WaitForAnswer {
		t.Fatalf("re-selecting the same subject must not reset in-progress state, got %v", s.State())
	}

	if !s.DiagnosisComplete() {
		t.Fatal("re-selecting the same subject must not reset diagnosis")
	}

	if s.CurrentQuestion() != "What is 7 + 3?" {
		t.Fatalf("re-selecting the same subject must not clear the pending question, got %q", s.CurrentQuestion())
	}
}

// TestMasteryIsPreservedAcrossSubjectSwitch is a direct test of the
// statistical-analysis prerequisite: mastery must be tracked per subject,
// not as a single session-wide flag, so switching subjects and back does
// not erase what the learner already demonstrated.
func TestMasteryIsPreservedAcrossSubjectSwitch(t *testing.T) {
	s := session.New("Amaka")

	s.SelectSubject("Mathematics")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")
	s.SubmitAnswer("10")
	s.CheckAnswer()
	s.AskTransferQuestion("What is 5 + 6?", "11")
	s.SubmitAnswer("11")
	s.CheckAnswer()

	if !s.IsMastered() {
		t.Fatal("expected Mathematics to be mastered before switching subjects")
	}

	s.SelectSubject("Christian Religious Studies")

	if s.IsMastered() {
		t.Fatal("a freshly selected subject must not inherit mastery from the previous subject")
	}

	s.SelectSubject("Mathematics")

	if !s.IsMastered() {
		t.Fatal("switching back to Mathematics must restore its previously recorded mastery")
	}

	if s.State() != session.Mastered {
		t.Fatalf("switching back to a mastered subject should restore state Mastered, got %v", s.State())
	}
}

// TestAdvanceTurnMovesToNextLearnerInOrder is a direct test of README
// §3 "One active respondent" / Blueprint lesson #1 "explicit turn
// management" for the MOVE_TO_NEXT_LEARNER_OR_QUESTION stage of the
// state machine (README §9): the roster must be able to move the turn
// forward, in roster order, so a different learner becomes active.
func TestAdvanceTurnMovesToNextLearnerInOrder(t *testing.T) {
	r := session.NewRoster("Amaka", "Bello", "Chidi")

	if r.ActiveLearner() != "Amaka" {
		t.Fatalf("expected Amaka to start active, got %s", r.ActiveLearner())
	}

	r.AdvanceTurn()
	if r.ActiveLearner() != "Bello" {
		t.Fatalf("expected Bello to become active, got %s", r.ActiveLearner())
	}

	r.AdvanceTurn()
	if r.ActiveLearner() != "Chidi" {
		t.Fatalf("expected Chidi to become active, got %s", r.ActiveLearner())
	}
}

// TestAdvanceTurnWrapsAroundToFirstLearner confirms turn order is a
// rotation, not a one-shot sequence — after the last learner, the turn
// returns to the first so the tutoring loop can continue indefinitely.
func TestAdvanceTurnWrapsAroundToFirstLearner(t *testing.T) {
	r := session.NewRoster("Amaka", "Bello")

	r.AdvanceTurn()
	if r.ActiveLearner() != "Bello" {
		t.Fatalf("expected Bello to become active, got %s", r.ActiveLearner())
	}

	r.AdvanceTurn()
	if r.ActiveLearner() != "Amaka" {
		t.Fatalf("expected turn to wrap around to Amaka, got %s", r.ActiveLearner())
	}
}

// TestAdvanceTurnOnSingleLearnerRosterIsANoOp confirms a roster of one
// learner does not lose its active learner when advanced — there is no
// one else for the turn to move to.
func TestAdvanceTurnOnSingleLearnerRosterIsANoOp(t *testing.T) {
	r := session.NewRoster("Amaka")

	r.AdvanceTurn()

	if r.ActiveLearner() != "Amaka" {
		t.Fatalf("expected Amaka to remain active on a single-learner roster, got %s", r.ActiveLearner())
	}
}

// TestSwitchingAwayFromUnmasteredProgressDiscardsInProgressState covers
// the subject-switching edge case where a learner has answered the
// original question correctly but has not yet passed the transfer/
// verification question (README §3, "Correct ≠ understanding": a
// correct answer alone is not durable evidence of mastery). Switching
// subjects and back must not resurrect that in-progress, unverified
// state — the learner must be re-diagnosed, same as any other
// non-mastered subject.
func TestSwitchingAwayFromUnmasteredProgressDiscardsInProgressState(t *testing.T) {
	s := session.New("Amaka")

	s.SelectSubject("Mathematics")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")
	s.SubmitAnswer("10")
	s.CheckAnswer()

	if s.State() != session.AwaitingTransferCheck {
		t.Fatalf("expected AwaitingTransferCheck before switching, got %v", s.State())
	}

	s.SelectSubject("Christian Religious Studies")
	s.SelectSubject("Mathematics")

	if s.IsMastered() {
		t.Fatal("a correct-but-unverified answer must not count as mastery after switching back")
	}

	if s.State() != session.Idle {
		t.Fatalf("expected state Idle after returning to an unmastered subject, got %v", s.State())
	}

	if s.DiagnosisComplete() {
		t.Fatal("returning to an unmastered subject must require diagnosis again")
	}

	if s.CurrentQuestion() != "" {
		t.Fatalf("returning to an unmastered subject must not resurrect the old pending question, got %q", s.CurrentQuestion())
	}

	if s.HasAnswer() {
		t.Fatal("returning to an unmastered subject must not resurrect the old answer")
	}

	err := s.AskQuestion("What is 7 + 3?", "10")
	if err == nil {
		t.Fatal("AskQuestion must be rejected until diagnosis is redone for this subject")
	}
}

// TestSubmitAnswerDoesNotAutomaticallyAdvanceTurn confirms turn
// advancement is not an automatic side effect of checking an answer.
// Per product decision, the roster's active learner only changes when a
// caller (the teacher/orchestrator) explicitly calls AdvanceTurn — never
// as an implicit consequence of SubmitAnswerFor or CheckAnswer, even
// when the answer is correct and mastery is recorded.
func TestSubmitAnswerDoesNotAutomaticallyAdvanceTurn(t *testing.T) {
	r := session.NewRoster("Amaka", "Bello")

	amaka := r.Session("Amaka")
	amaka.SelectSubject("Mathematics")
	amaka.StartDiagnosis("What do you already know about addition?")
	amaka.RecordDiagnosticResponse("I know how to add single digits")
	amaka.AskQuestion("What is 7 + 3?", "10")

	if err := r.SubmitAnswerFor("Amaka", "10"); err != nil {
		t.Fatalf("expected SubmitAnswerFor to succeed, got error: %v", err)
	}
	amaka.CheckAnswer()
	amaka.AskTransferQuestion("What is 5 + 6?", "11")

	if err := r.SubmitAnswerFor("Amaka", "11"); err != nil {
		t.Fatalf("expected SubmitAnswerFor to succeed, got error: %v", err)
	}
	amaka.CheckAnswer()

	if !amaka.IsMastered() {
		t.Fatal("expected Amaka's Mathematics to be mastered at this point")
	}

	if r.ActiveLearner() != "Amaka" {
		t.Fatalf("turn must not auto-advance on mastery; expected Amaka still active, got %s", r.ActiveLearner())
	}
}

// TestTopicDefaultsEmpty confirms a new session has no topic selected
// until SelectTopic is called (README §6, SELECT_SUBJECT -> SELECT_TOPIC).
func TestTopicDefaultsEmpty(t *testing.T) {
	s := session.New("Amaka")
	if s.Topic() != "" {
		t.Fatalf("expected empty topic before SelectTopic, got %q", s.Topic())
	}
}

// TestSelectTopicRequiresSubjectSelectedFirst enforces README §6's
// nesting: SELECT_SUBJECT must happen before SELECT_TOPIC. Selecting a
// topic before any subject is chosen is rejected rather than silently
// allowed.
func TestSelectTopicRequiresSubjectSelectedFirst(t *testing.T) {
	s := session.New("Amaka")

	err := s.SelectTopic("Addition")
	if err == nil {
		t.Fatal("expected SelectTopic to be rejected before a subject is selected")
	}

	if s.Topic() != "" {
		t.Fatalf("rejected SelectTopic must not change the topic, got %q", s.Topic())
	}
}

// TestSwitchingTopicResetsDiagnosisState mirrors subject switching one
// level deeper: switching to a genuinely different topic within the same
// subject must reset diagnosis and clear any pending question, answer,
// and confirmation state, since the new topic has not been diagnosed yet.
func TestSwitchingTopicResetsDiagnosisState(t *testing.T) {
	s := session.New("Amaka")
	s.SelectSubject("Mathematics")
	if err := s.SelectTopic("Addition"); err != nil {
		t.Fatalf("expected SelectTopic to succeed, got error: %v", err)
	}
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")

	if err := s.SelectTopic("Fractions"); err != nil {
		t.Fatalf("expected SelectTopic to succeed, got error: %v", err)
	}

	if s.Topic() != "Fractions" {
		t.Fatalf("expected topic to update, got %q", s.Topic())
	}

	if s.DiagnosisComplete() {
		t.Fatal("switching topics must reset diagnosis for the new topic")
	}

	if s.State() != session.Idle {
		t.Fatalf("expected state Idle after switching topics, got %v", s.State())
	}

	if s.CurrentQuestion() != "" {
		t.Fatalf("switching topics must clear the pending question, got %q", s.CurrentQuestion())
	}

	if s.ConfirmationCount() != 0 {
		t.Fatalf("switching topics must reset confirmation count, got %d", s.ConfirmationCount())
	}

	err := s.AskQuestion("What is 1/2 + 1/4?", "3/4")
	if err == nil {
		t.Fatal("AskQuestion must be rejected until the new topic has been diagnosed")
	}
}

// TestSelectingSameTopicDoesNotResetState confirms SelectTopic is
// idempotent for the current topic, same as SelectSubject.
func TestSelectingSameTopicDoesNotResetState(t *testing.T) {
	s := session.New("Amaka")
	s.SelectSubject("Mathematics")
	s.SelectTopic("Addition")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")

	if err := s.SelectTopic("Addition"); err != nil {
		t.Fatalf("expected re-selecting the same topic to succeed, got error: %v", err)
	}

	if s.State() != session.WaitForAnswer {
		t.Fatalf("re-selecting the same topic must not reset in-progress state, got %v", s.State())
	}

	if s.CurrentQuestion() != "What is 7 + 3?" {
		t.Fatalf("re-selecting the same topic must not clear the pending question, got %q", s.CurrentQuestion())
	}
}

// TestMasteryIsTrackedPerTopicNotPerSubject confirms mastery granularity:
// mastering one topic within a subject must not mark a different topic
// in the same subject as mastered, and switching between topics must
// preserve each topic's own mastery independently.
func TestMasteryIsTrackedPerTopicNotPerSubject(t *testing.T) {
	s := session.New("Amaka")

	s.SelectSubject("Mathematics")
	s.SelectTopic("Addition")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")
	s.SubmitAnswer("10")
	s.CheckAnswer()
	s.AskTransferQuestion("What is 5 + 6?", "11")
	s.SubmitAnswer("11")
	s.CheckAnswer()

	if !s.IsMastered() {
		t.Fatal("expected Addition to be mastered before switching topics")
	}

	s.SelectTopic("Fractions")

	if s.IsMastered() {
		t.Fatal("a freshly selected topic must not inherit mastery from a different topic in the same subject")
	}

	s.SelectTopic("Addition")

	if !s.IsMastered() {
		t.Fatal("switching back to Addition must restore its previously recorded mastery")
	}

	if s.State() != session.Mastered {
		t.Fatalf("switching back to a mastered topic should restore state Mastered, got %v", s.State())
	}
}

// TestSwitchingSubjectResetsTopicSelection confirms a subject switch
// clears the topic — a new subject has not had any topic chosen within
// it yet, so the previous subject's topic must not carry over.
func TestSwitchingSubjectResetsTopicSelection(t *testing.T) {
	s := session.New("Amaka")
	s.SelectSubject("Mathematics")
	s.SelectTopic("Addition")

	s.SelectSubject("Christian Religious Studies")

	if s.Topic() != "" {
		t.Fatalf("switching subjects must clear the previously selected topic, got %q", s.Topic())
	}
}

// TestResumingSessionDoesNotRepeatCompletedDiagnosis is a direct test of
// README §6: "must support resuming an unfinished session without
// repeating the diagnostic stage from scratch." If an interruption or
// resumption path re-invokes StartDiagnosis on a session whose diagnosis
// is already complete for the current subject/topic, it must not discard
// the existing diagnostic finding or any in-progress question state.
func TestResumingSessionDoesNotRepeatCompletedDiagnosis(t *testing.T) {
	s := session.New("Amaka")
	s.SelectSubject("Mathematics")
	s.SelectTopic("Addition")
	s.StartDiagnosis("What do you already know about addition?")
	s.RecordDiagnosticResponse("I know how to add single digits")
	s.AskQuestion("What is 7 + 3?", "10")

	// Simulate an interruption/resumption path re-entering diagnosis.
	s.StartDiagnosis("What do you already know about addition?")

	if !s.DiagnosisComplete() {
		t.Fatal("resuming must not re-open diagnosis that was already complete")
	}

	if s.DiagnosticFinding() != "I know how to add single digits" {
		t.Fatalf("resuming must not discard the recorded diagnostic finding, got %q", s.DiagnosticFinding())
	}

	if s.State() != session.WaitForAnswer {
		t.Fatalf("resuming must not disturb in-progress state, got %v", s.State())
	}

	if s.CurrentQuestion() != "What is 7 + 3?" {
		t.Fatalf("resuming must not clear the pending question, got %q", s.CurrentQuestion())
	}
}

// TestStartDiagnosisStillWorksWhenNotYetComplete confirms the resumption
// guard only protects an already-completed diagnosis — starting
// diagnosis for the first time, or restarting it while still mid-
// diagnosis (not yet complete), must still work normally.
func TestStartDiagnosisStillWorksWhenNotYetComplete(t *testing.T) {
	s := session.New("Amaka")
	s.SelectSubject("Mathematics")
	s.SelectTopic("Addition")
	s.StartDiagnosis("What do you already know about addition?")

	if s.State() != session.Diagnosing {
		t.Fatalf("expected state Diagnosing, got %v", s.State())
	}

	if s.DiagnosisComplete() {
		t.Fatal("diagnosis must not be marked complete before RecordDiagnosticResponse is called")
	}
}

// TestGradeBandDefaultsEmpty confirms a new session has no grade band
// until SetGradeBand is called (README §10, per-learner data must
// include grade band).
func TestGradeBandDefaultsEmpty(t *testing.T) {
	s := session.New("Amaka")
	if s.GradeBand() != "" {
		t.Fatalf("expected empty grade band before SetGradeBand, got %q", s.GradeBand())
	}
}

// TestSetGradeBandAcceptsOnlyDefinedBands enforces README §4's
// non-negotiable age-band calibration requirement deterministically:
// only the three defined bands (Primary 4-6, JSS1-3, SSS1-3) are valid.
// An undefined band must be rejected rather than silently accepted,
// since age calibration is a safety requirement, not cosmetic.
func TestSetGradeBandAcceptsOnlyDefinedBands(t *testing.T) {
	s := session.New("Amaka")

	if err := s.SetGradeBand("JSS1-3"); err != nil {
		t.Fatalf("expected a defined grade band to be accepted, got error: %v", err)
	}
	if s.GradeBand() != "JSS1-3" {
		t.Fatalf("expected grade band to be set, got %q", s.GradeBand())
	}

	err := s.SetGradeBand("Grade 9")
	if err == nil {
		t.Fatal("expected an undefined grade band to be rejected")
	}
	if s.GradeBand() != "JSS1-3" {
		t.Fatalf("a rejected grade band must not overwrite the existing one, got %q", s.GradeBand())
	}
}

// TestGradeBandSurvivesSubjectAndTopicSwitch confirms grade band is
// learner-level state (README §10), not subject- or topic-level, so it
// must not be cleared by SelectSubject or SelectTopic the way diagnosis
// and pending question state are.
func TestGradeBandSurvivesSubjectAndTopicSwitch(t *testing.T) {
	s := session.New("Amaka")
	s.SetGradeBand("Primary 4-6")

	s.SelectSubject("Mathematics")
	s.SelectTopic("Addition")
	s.SelectSubject("Christian Religious Studies")

	if s.GradeBand() != "Primary 4-6" {
		t.Fatalf("grade band must survive subject/topic switches, got %q", s.GradeBand())
	}
}
