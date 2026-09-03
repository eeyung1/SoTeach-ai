package tutor

import (
	"errors"

	"soteach/session"
)

// ErrSessionNotAwaitingDiagnosis is returned when a diagnostic explanation is
// submitted for a session that is not in the diagnostic stage — e.g. one that
// has already moved on to a pending question. Submitting again would repeat
// the diagnostic stage (README §6) or clobber a recorded finding.
var ErrSessionNotAwaitingDiagnosis = errors.New("session is not awaiting a diagnostic explanation")

// ErrTopicNotYetSupported is returned when no practice content exists yet for
// the session's topic. The MVP deliberately starts with a narrow concept set
// (README §5); content is added one supported topic at a time.
var ErrTopicNotYetSupported = errors.New("no practice content yet for this topic")

// Tutor is the application layer that owns the tutoring loop (README §2;
// Agent.md §11): it decides what happens next in a learner's session and what
// the learner is asked or told. A thin client never sequences the loop — it
// sends the learner's input and receives the next prompt.
//
// Question and explanation content is served by the Tutor itself (a
// deterministic stub for now, to be replaced by AI-generated content when a
// concrete requirement exists — Agent.md §38).
type Tutor struct {
	store *session.MemoryStore
}

// NewTutor creates a Tutor over the given session store.
func NewTutor(store *session.MemoryStore) *Tutor {
	return &Tutor{store: store}
}

// Outcome describes what the tutor says to the learner after an action.
type Outcome struct {
	// Prompt is the next message/question to show the learner.
	Prompt string
}

// SubmitDiagnosis records the learner's diagnostic explanation and, because
// the engine owns the loop, immediately poses the first practice question
// from server-side content. The session is saved at the resulting state
// (WaitForAnswer with the first question pending). It is rejected if the
// session is not currently awaiting a diagnostic explanation.
func (t *Tutor) SubmitDiagnosis(learner, explanation string) (Outcome, error) {
	s, err := t.store.Load(learner)
	if err != nil {
		return Outcome{}, err
	}

	if s.State() != session.Diagnosing || s.DiagnosisComplete() {
		return Outcome{}, ErrSessionNotAwaitingDiagnosis
	}

	question, expected, err := firstPracticeQuestion(s.Topic())
	if err != nil {
		return Outcome{}, err
	}

	s.RecordDiagnosticResponse(explanation)
	if err := s.AskQuestion(question, expected); err != nil {
		return Outcome{}, err
	}
	t.store.Save(s)

	return Outcome{Prompt: question}, nil
}

// firstPracticeQuestion is the server-owned content source for the MVP: the
// first practice question a learner gets for a topic, with its deterministic
// expected answer. Content arrives one supported topic at a time (README §5).
func firstPracticeQuestion(topic string) (question, expectedAnswer string, err error) {
	switch topic {
	case "Addition":
		return "What is 7 + 3?", "10", nil
	default:
		return "", "", ErrTopicNotYetSupported
	}
}
