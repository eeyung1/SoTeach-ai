package session

import (
	"errors"
	"strings"

	"soteach/ai"
)

// State represents a step in the tutoring state machine (README §6).
// Only the states required by the current behavior are defined here.
type State int

const (
	Idle State = iota
	Diagnosing
	WaitForAnswer
	CheckAnswer
	AwaitingTransferCheck
	Incorrect
	Uncertain
	Mastered
)

// maxConfirmations is the exact number of confirmations allowed per answer
// (README §3, rule "Confirmation count is exact"). This must never be
// left to the LLM's own conversational memory — it is enforced here as
// explicit state.
const maxConfirmations = 2

// ErrDiagnosisIncomplete is returned when TEACH/PRACTICE is attempted
// before DIAGNOSE has recorded a finding (README §2: "No stage is
// optional, and no stage may be silently skipped by the model.").
var ErrDiagnosisIncomplete = errors.New("diagnosis must be completed before asking a question")

// ErrQuestionAlreadyPending is returned when AskQuestion is called while a
// question is already awaiting an answer (README §3, rule "Wait for the
// learner": "Never advance to the next question while one is pending.").
var ErrQuestionAlreadyPending = errors.New("cannot ask a new question while one is already pending")

// ErrTransferQuestionNotDistinct is returned when a transfer/verification
// question repeats the original question verbatim (README §2, Stage 4:
// VERIFY must use "a genuinely different question, not a repeat of the
// same one").
var ErrTransferQuestionNotDistinct = errors.New("transfer question must be genuinely different from the original question")

// ErrReteachingRequired is returned when AskQuestion is called after an
// uncertain answer ("I don't know") without an intervening Teach() call.
// Product decision: uncertainty must route toward re-teaching, not
// straight back into another practice question — unlike a genuine
// wrong-but-attempted (Incorrect) answer, which may loop back directly.
var ErrReteachingRequired = errors.New("must teach the gap before asking another question after an uncertain answer")

// ErrSubjectNotSelected is returned when SelectTopic is called before a
// subject has been selected (README §6, SELECT_SUBJECT -> SELECT_TOPIC:
// topic selection is nested inside subject selection, not independent
// of it).
var ErrSubjectNotSelected = errors.New("a subject must be selected before a topic can be selected")

// validGradeBands are the three age bands defined by README §4 (age-band
// calibration is a non-negotiable product and safety requirement, not
// cosmetic) — Primary 4-6, JSS1-3, SSS1-3. No other value is accepted.
var validGradeBands = map[string]bool{
	"Primary 4-6": true,
	"JSS1-3":      true,
	"SSS1-3":      true,
}

// ErrInvalidGradeBand is returned when SetGradeBand is called with a
// value outside the three defined age bands.
var ErrInvalidGradeBand = errors.New("grade band must be one of: Primary 4-6, JSS1-3, SSS1-3")

// GradeBands returns the three defined grade/age bands (README §4) in a stable
// order, so a server can offer them to a client and validate a learner's band
// against a single source of truth.
func GradeBands() []string {
	return []string{"Primary 4-6", "JSS1-3", "SSS1-3"}
}

// Session holds the minimal state needed to satisfy the "wait for the
// student" and "never mark correct without checking" rules (README §3).
//
// A Session belongs to exactly one learner (README §3, rule "Multiple
// learners, separate state"). Two learners in the same tutoring context
// are represented as two independent Session values — nothing here merges
// or shares state across learners.
type Session struct {
	learnerName          string
	gradeBand            string
	subject              string
	topic                string
	state                State
	question             string
	expectedAnswer       string
	answer               string
	hasAnswer            bool
	mastered             bool
	masteryBySubject     map[string]bool
	confirmationCount    int
	diagnosticPrompt     string
	diagnosticFinding    string
	diagnosticResult     ai.DiagnosticResult
	diagnosisComplete    bool
	originalQuestion     string
	isTransferQuestion   bool
	taughtSinceUncertain bool
	lastTeaching         string
	wrongStreak          int
}

// New creates a session for a single named learner.
func New(learnerName string) *Session {
	return &Session{learnerName: learnerName, state: Idle, masteryBySubject: map[string]bool{}}
}

// LearnerName returns the name of the learner this session belongs to.
func (s *Session) LearnerName() string {
	return s.learnerName
}

// GradeBand returns the learner's currently set age/grade band, or the
// empty string if none has been set yet.
func (s *Session) GradeBand() string {
	return s.gradeBand
}

// SetGradeBand sets the learner's age/grade band (README §4, §10). This
// is learner-level state, not subject- or topic-level: it is deliberately
// untouched by SelectSubject/SelectTopic, unlike diagnosis and pending
// question state. Only the three defined bands are accepted; an
// undefined value is rejected and does not overwrite whatever was
// already set.
func (s *Session) SetGradeBand(band string) error {
	if !validGradeBands[band] {
		return ErrInvalidGradeBand
	}
	s.gradeBand = band
	return nil
}

// Subject returns the currently selected subject, or the empty string if
// no subject has been selected yet.
func (s *Session) Subject() string {
	return s.subject
}

// SelectSubject sets the active subject (Blueprint lesson #9: "The system
// must handle subject switching, such as Mathematics to Christian
// Religious Studies").
//
// Selecting a genuinely different subject resets diagnosis and clears any
// pending question, answer, and confirmation state left over from the
// previous subject — the new subject has not been diagnosed yet, and none
// of that state belongs to it. Re-selecting the subject already in
// progress is a no-op with respect to that state, so it does not destroy
// work in progress.
func (s *Session) SelectSubject(subject string) {
	if subject == s.subject {
		return
	}

	s.subject = subject
	s.topic = ""
	s.resetForNewSelection()
	s.restoreMasteryForCurrentSelection()
}

// SelectTopic sets the active topic within the currently selected subject
// (README §6, SELECT_SUBJECT -> SELECT_TOPIC). It is rejected until a
// subject has been selected.
//
// Selecting a genuinely different topic resets diagnosis and clears any
// pending question, answer, and confirmation state, the same way
// SelectSubject does — the new topic has not been diagnosed yet.
// Re-selecting the topic already in progress is a no-op with respect to
// that state.
func (s *Session) SelectTopic(topic string) error {
	if s.subject == "" {
		return ErrSubjectNotSelected
	}

	if topic == s.topic {
		return nil
	}

	s.topic = topic
	s.resetForNewSelection()
	s.restoreMasteryForCurrentSelection()
	return nil
}

// Topic returns the currently selected topic, or the empty string if no
// topic has been selected yet.
func (s *Session) Topic() string {
	return s.topic
}

// resetForNewSelection clears diagnosis and pending question/answer/
// confirmation state shared by SelectSubject and SelectTopic when moving
// to a genuinely new subject or topic. It does not touch mastery — the
// caller restores that afterward via restoreMasteryForCurrentSelection.
func (s *Session) resetForNewSelection() {
	s.state = Idle
	s.diagnosisComplete = false
	s.diagnosticPrompt = ""
	s.diagnosticFinding = ""
	s.diagnosticResult = ai.DiagnosticResult{}
	s.question = ""
	s.originalQuestion = ""
	s.expectedAnswer = ""
	s.answer = ""
	s.hasAnswer = false
	s.confirmationCount = 0
	s.isTransferQuestion = false
	s.taughtSinceUncertain = false
	s.lastTeaching = ""
	s.wrongStreak = 0
}

// masteryKey identifies the current subject+topic for mastery tracking.
// Mastery is tracked per topic within a subject (not per subject as a
// whole), so mastering Mathematics/Addition must not mark
// Mathematics/Fractions as mastered.
func (s *Session) masteryKey() string {
	return s.subject + "|" + s.topic
}

// restoreMasteryForCurrentSelection restores mastered/state from the
// per-subject-and-topic mastery map for whatever subject+topic is
// currently selected, so switching away and back does not erase
// previously demonstrated mastery.
func (s *Session) restoreMasteryForCurrentSelection() {
	if s.masteryBySubject[s.masteryKey()] {
		s.mastered = true
		s.state = Mastered
	} else {
		s.mastered = false
	}
}

// ResetAttempt clears the in-progress diagnosis/question/answer state for the
// current subject/topic, so a learner who explicitly stops can begin again
// from scratch — while keeping learner-level state (name, grade band) and any
// previously demonstrated mastery intact.
func (s *Session) ResetAttempt() {
	s.resetForNewSelection()
	s.restoreMasteryForCurrentSelection()
}

// StartDiagnosis begins README §2 Stage 1 (DIAGNOSE): asking the learner
// to explain what they already know before any teaching happens. This
// must happen before TEACH, not be skipped or assumed.
//
// If diagnosis for the current subject/topic is already complete,
// StartDiagnosis is a no-op (README §6: resuming an unfinished session
// must not repeat the diagnostic stage from scratch). An interruption or
// resumption path may re-invoke StartDiagnosis without realizing
// diagnosis already happened; this guard protects the existing
// diagnostic finding and any in-progress question state from being
// silently discarded.
func (s *Session) StartDiagnosis(prompt string) {
	if s.diagnosisComplete {
		return
	}

	s.diagnosticPrompt = prompt
	s.diagnosisComplete = false
	s.state = Diagnosing
}

// RecordDiagnosticResponse records the learner's own explanation of what
// they already know, verbatim, so the actual gap can be located instead
// of assumed (README §2, Stage 1). Recording a response marks diagnosis
// complete.
func (s *Session) RecordDiagnosticResponse(response string) {
	s.diagnosticFinding = response
	s.diagnosisComplete = true
}

// DiagnosisComplete reports whether the learner's diagnostic explanation
// has been recorded for the current concept.
func (s *Session) DiagnosisComplete() bool {
	return s.diagnosisComplete
}

// DiagnosticFinding returns the learner's own recorded explanation from
// the diagnostic stage.
func (s *Session) DiagnosticFinding() string {
	return s.diagnosticFinding
}

// RecordDiagnosticResult records an AI-validated diagnosis (Agent.md
// §39-40) as an alternative to RecordDiagnosticResponse. The result is
// validated at this boundary, where state actually changes, before it is
// allowed to mark diagnosis complete — a malformed result is rejected
// outright and leaves diagnosisComplete and the stored result untouched.
func (s *Session) RecordDiagnosticResult(result ai.DiagnosticResult) error {
	if err := ai.ValidateDiagnosticResult(result); err != nil {
		return err
	}

	s.diagnosticResult = result
	s.diagnosisComplete = true
	return nil
}

// LastDiagnosticResult returns the most recently recorded AI-validated
// diagnostic result, or the zero value if none has been recorded (or it
// has been cleared by a subject/topic switch).
func (s *Session) LastDiagnosticResult() ai.DiagnosticResult {
	return s.diagnosticResult
}

// AskQuestion poses the original practice question along with its expected
// answer, and moves the session into WaitForAnswer. The expected answer is
// required so that checking can be deterministic (README §8.4) rather than
// left to model judgment.
//
// AskQuestion is rejected until DIAGNOSE has recorded a finding (README
// §2: no stage may be silently skipped), and rejected while a question is
// already pending (README §3, rule "Wait for the learner"). On rejection,
// state and the pending question are left completely unchanged.
func (s *Session) AskQuestion(question, expectedAnswer string) error {
	if !s.diagnosisComplete {
		return ErrDiagnosisIncomplete
	}

	if s.state == WaitForAnswer {
		return ErrQuestionAlreadyPending
	}

	if s.state == Uncertain && !s.taughtSinceUncertain {
		return ErrReteachingRequired
	}

	s.question = question
	s.originalQuestion = question
	s.expectedAnswer = expectedAnswer
	s.isTransferQuestion = false
	s.state = WaitForAnswer
	s.hasAnswer = false
	s.confirmationCount = 0
	s.taughtSinceUncertain = false
	return nil
}

// AskTransferQuestion poses a verification question after an initial
// correct answer (README §2, Stage 4 VERIFY; README §3, rule "Correct ≠
// understanding"). It must be genuinely different from the original
// question — repeating the same question is not acceptable evidence of
// understanding.
func (s *Session) AskTransferQuestion(question, expectedAnswer string) error {
	if question == s.originalQuestion {
		return ErrTransferQuestionNotDistinct
	}

	s.question = question
	s.expectedAnswer = expectedAnswer
	s.isTransferQuestion = true
	s.state = WaitForAnswer
	s.hasAnswer = false
	s.confirmationCount = 0
	return nil
}

// SubmitAnswer records the learner's answer and moves the session into
// CheckAnswer. It deliberately does NOT decide correctness (README §8.4) —
// that is CheckAnswer's job.
func (s *Session) SubmitAnswer(a string) {
	s.answer = a
	s.hasAnswer = true
	s.state = CheckAnswer
}

// CheckAnswer deterministically compares the submitted answer against the
// expected answer. This is the direct fix for the observed failure in
// README §3 where "7 + 3" was wrongly accepted. No AI is involved in this
// decision.
//
// A correct answer to the original question does NOT record mastery
// (README §3, rule "Correct ≠ understanding"): guessing, memorizing, or
// copying can also produce a correct answer. The session instead moves to
// AwaitingTransferCheck, which requires a transfer/verification question
// (see AskTransferQuestion) before mastery can be recorded. Only a correct
// answer to that transfer question sets Mastered.
// uncertainPhrases are common ways a learner signals they don't know the
// answer rather than making a genuine incorrect attempt (README §9,
// "explicit uncertainty behavior").
var uncertainPhrases = map[string]bool{
	"i don't know": true,
	"i dont know":  true,
	"idk":          true,
	"not sure":     true,
	"no idea":      true,
}

func (s *Session) CheckAnswer() bool {
	trimmedAnswer := strings.TrimSpace(s.answer)

	if uncertainPhrases[strings.ToLower(trimmedAnswer)] {
		s.state = Uncertain
		return false
	}

	correct := strings.EqualFold(
		trimmedAnswer,
		strings.TrimSpace(s.expectedAnswer),
	)

	switch {
	case correct && s.isTransferQuestion:
		s.mastered = true
		s.state = Mastered
		if s.masteryBySubject == nil {
			s.masteryBySubject = map[string]bool{}
		}
		if s.subject != "" {
			s.masteryBySubject[s.masteryKey()] = true
		}
	case correct:
		s.state = AwaitingTransferCheck
	default:
		s.state = Incorrect
	}

	return correct
}

// RequestConfirmation records one confirmation request against the exact
// confirmation-count rule (README §3, README §8.3): exactly two are
// allowed per answer. It returns true if the request was allowed, false
// if the limit has already been reached. The count is explicit state on
// Session, not left to an LLM's own memory of how many times it has asked.
func (s *Session) RequestConfirmation() bool {
	if s.state == Idle || s.state == Diagnosing || s.state == WaitForAnswer {
		return false
	}

	if s.confirmationCount >= maxConfirmations {
		return false
	}
	s.confirmationCount++
	return true
}

// ConfirmationCount reports how many confirmations have been recorded for
// the current answer.
func (s *Session) ConfirmationCount() int {
	return s.confirmationCount
}

func (s *Session) State() State {
	return s.state
}

func (s *Session) HasAnswer() bool {
	return s.hasAnswer
}

// IsMastered reports whether the session has recorded mastery of the
// current concept. Mastery is never set by a correct answer to the
// original question alone — it can only be set once a transfer/
// verification question is also answered correctly.
func (s *Session) IsMastered() bool {
	return s.mastered
}

// WrongStreak returns how many consecutive practice answers have been
// incorrect. It is reset by a correct answer, an uncertain answer, teaching,
// or moving to a new subject/topic. The tutor uses it to stop endlessly
// re-asking and teach the gap instead (README §9).
func (s *Session) WrongStreak() int {
	return s.wrongStreak
}

// NoteIncorrectAttempt increments the consecutive-wrong count.
func (s *Session) NoteIncorrectAttempt() {
	s.wrongStreak++
}

// ResetWrongStreak clears the consecutive-wrong count.
func (s *Session) ResetWrongStreak() {
	s.wrongStreak = 0
}

// CurrentQuestion returns the text of the currently pending question so a
// learner interruption like "what was the question?" (README §3, "handle
// interruptions without losing state") can be answered without affecting
// state, the recorded answer, or the confirmation count. Safe to call
// repeatedly and idempotently.
func (s *Session) CurrentQuestion() string {
	return s.question
}

// Teach records that the learner's gap has been explained following an
// uncertain answer (README §2, TEACH stage). It must be called before
// AskQuestion is allowed to proceed again when the session is in the
// Uncertain state.
func (s *Session) Teach(explanation string) {
	s.lastTeaching = explanation

	if s.state == Uncertain {
		s.taughtSinceUncertain = true
	}
}

// LastTeaching returns the explanation given by the most recent call to
// Teach, or an empty string if Teach has never been called (README §9
// safeguard 6, auditability; README §10, diagnostic findings).
func (s *Session) LastTeaching() string {
	return s.lastTeaching
}
