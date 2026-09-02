package session

import "strings"

// State represents a step in the tutoring state machine (README §6).
// Only the states required by the current behavior are defined here.
type State int

const (
	Idle State = iota
	WaitForAnswer
	CheckAnswer
	Correct
	Incorrect
)

// Session holds the minimal state needed to satisfy the "wait for the
// student" and "never mark correct without checking" rules (README §3).
type Session struct {
	state           State
	question        string
	expectedAnswer  string
	answer          string
	hasAnswer       bool
}

func New() *Session {
	return &Session{state: Idle}
}

// AskQuestion poses a question along with its expected answer, and moves
// the session into WaitForAnswer. The expected answer is required so that
// checking can be deterministic (README §8.4) rather than left to model
// judgment.
func (s *Session) AskQuestion(question, expectedAnswer string) {
	s.question = question
	s.expectedAnswer = expectedAnswer
	s.state = WaitForAnswer
	s.hasAnswer = false
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
func (s *Session) CheckAnswer() bool {
	correct := strings.EqualFold(
		strings.TrimSpace(s.answer),
		strings.TrimSpace(s.expectedAnswer),
	)

	if correct {
		s.state = Correct
	} else {
		s.state = Incorrect
	}

	return correct
}

func (s *Session) State() State {
	return s.state
}

func (s *Session) HasAnswer() bool {
	return s.hasAnswer
}
