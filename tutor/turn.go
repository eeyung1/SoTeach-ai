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

// ErrNoQuestionPending is returned when an answer is submitted while no
// question is awaiting one (README §3, rule "Wait for the learner" — there is
// nothing to answer yet).
var ErrNoQuestionPending = errors.New("no question is pending for this learner")

// ErrUnexpectedAnswerState is returned if an answer routes to a session state
// the tutor does not know how to continue from. It indicates a bug rather
// than a normal learner condition.
var ErrUnexpectedAnswerState = errors.New("tutor does not know how to continue from this answer state")

// ErrNothingAwaitingInput is returned when learner input arrives at a point
// where the session is not waiting for any — e.g. after mastery, when there
// is nothing left to answer in this loop (README §3). It must not be silently
// ignored or misrouted.
var ErrNothingAwaitingInput = errors.New("no input is expected from the learner right now")

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

// SubmitAnswer routes the learner's answer to the pending question according
// to the tutoring loop (README §2):
//
//   - correct answer to the original practice question -> the server poses a
//     genuinely different transfer question (README §3, "Correct ≠
//     understanding"; §2 Stage 4 VERIFY)
//   - correct answer to that transfer question -> mastery is recorded and the
//     learner is told the concept is complete
//   - incorrect answer -> a fresh practice question for the same gap
//     (README §6, HANDLE_INCORRECT_OR_UNCERTAIN_ANSWER loops back)
//   - uncertain answer ("I don't know") -> the server teaches the gap first,
//     then asks a fresh question (README §9, explicit uncertainty behavior)
//
// It is rejected when no question is pending. The client supplies only the
// answer text; every routing and content decision stays server-side.
func (t *Tutor) SubmitAnswer(learner, answer string) (Outcome, error) {
	s, err := t.store.Load(learner)
	if err != nil {
		return Outcome{}, err
	}

	if s.State() != session.WaitForAnswer {
		return Outcome{}, ErrNoQuestionPending
	}

	s.SubmitAnswer(answer)
	s.CheckAnswer()

	switch s.State() {
	case session.Mastered:
		t.store.Save(s)
		return Outcome{Prompt: completionMessage(s.Topic())}, nil

	case session.AwaitingTransferCheck:
		question, expected, err := transferQuestion(s.Topic())
		if err != nil {
			return Outcome{}, err
		}
		if err := s.AskTransferQuestion(question, expected); err != nil {
			return Outcome{}, err
		}
		t.store.Save(s)
		return Outcome{Prompt: question}, nil

	case session.Incorrect:
		question, expected, err := nextPracticeQuestion(s.Topic(), s.CurrentQuestion())
		if err != nil {
			return Outcome{}, err
		}
		if err := s.AskQuestion(question, expected); err != nil {
			return Outcome{}, err
		}
		t.store.Save(s)
		return Outcome{Prompt: question}, nil

	case session.Uncertain:
		teach := explanation(s.Topic())
		question, expected, err := nextPracticeQuestion(s.Topic(), s.CurrentQuestion())
		if err != nil {
			return Outcome{}, err
		}
		s.Teach(teach)
		if err := s.AskQuestion(question, expected); err != nil {
			return Outcome{}, err
		}
		t.store.Save(s)
		return Outcome{Prompt: teach + " " + question}, nil

	default:
		return Outcome{}, ErrUnexpectedAnswerState
	}
}

// BeginTopic starts (or resumes) a learner's session on a topic — the
// server-side entry to the loop (README §6, SELECT_SUBJECT -> SELECT_TOPIC ->
// DIAGNOSE). A thin client never chooses the diagnostic wording; the server
// selects subject/topic and returns the first thing the learner should
// respond to.
//
// Beginning a topic the learner already has in progress resumes it at the
// correct state rather than repeating diagnosis (README §6): a pending
// question is returned, and already-recorded mastery is reported, not reset.
// A topic with no server-side content is rejected without creating a session
// (README §5).
func (t *Tutor) BeginTopic(learner, subject, topic string) (Outcome, error) {
	prompt, err := diagnosticPrompt(topic)
	if err != nil {
		return Outcome{}, err
	}

	s, err := t.store.Load(learner)
	if errors.Is(err, session.ErrSessionNotFound) {
		s = session.New(learner)
	} else if err != nil {
		return Outcome{}, err
	}

	s.SelectSubject(subject)
	if err := s.SelectTopic(topic); err != nil {
		return Outcome{}, err
	}

	var next string
	switch {
	case s.State() == session.Mastered:
		next = "You have already mastered " + topic + "."
	case s.State() == session.Idle:
		// Fresh topic (new learner, or switched from another topic): start
		// diagnosis for this topic.
		s.StartDiagnosis(prompt)
		next = prompt
	case !s.DiagnosisComplete():
		// Resumed mid-diagnosis: return the diagnostic prompt again.
		next = prompt
	default:
		// Diagnosis complete with a question pending: resume with that question.
		next = s.CurrentQuestion()
	}

	t.store.Save(s)
	return Outcome{Prompt: next}, nil
}

// ApplyInput routes one piece of learner input to the correct tutoring action
// from the session state alone (README §6): a learner mid-diagnosis is giving
// their explanation, while a learner with a question pending is giving an
// answer. Keeping this routing inside the engine means HTTP handlers never
// decide what an input means (Agent.md §19). It is rejected when no session
// exists for the learner, or when the session is not waiting for any input.
func (t *Tutor) ApplyInput(learner, input string) (Outcome, error) {
	s, err := t.store.Load(learner)
	if err != nil {
		return Outcome{}, err
	}

	switch {
	case s.State() == session.Diagnosing && !s.DiagnosisComplete():
		return t.SubmitDiagnosis(learner, input)
	case s.State() == session.WaitForAnswer:
		return t.SubmitAnswer(learner, input)
	default:
		return Outcome{}, ErrNothingAwaitingInput
	}
}

// qa pairs a question with its deterministic expected answer.
type qa struct {
	question       string
	expectedAnswer string
}

// additionPracticeQuestions is the ordered set of practice questions for the
// Addition topic. Content is deliberately small (README §5) and grows with
// the concept set.
var additionPracticeQuestions = []qa{
	{question: "What is 7 + 3?", expectedAnswer: "10"},
	{question: "What is 4 + 6?", expectedAnswer: "10"},
	{question: "What is 2 + 5?", expectedAnswer: "7"},
}

func practiceQuestions(topic string) ([]qa, error) {
	if topic != "Addition" {
		return nil, ErrTopicNotYetSupported
	}
	return additionPracticeQuestions, nil
}

// firstPracticeQuestion is the first practice question a learner gets for a
// topic, with its deterministic expected answer.
func firstPracticeQuestion(topic string) (question, expectedAnswer string, err error) {
	qs, err := practiceQuestions(topic)
	if err != nil {
		return "", "", err
	}
	return qs[0].question, qs[0].expectedAnswer, nil
}

// nextPracticeQuestion returns a practice question different from last for a
// topic, so an incorrect or uncertain answer loops back to a fresh question
// for the same gap rather than repeating the identical one.
func nextPracticeQuestion(topic, last string) (question, expectedAnswer string, err error) {
	qs, err := practiceQuestions(topic)
	if err != nil {
		return "", "", err
	}
	for _, q := range qs {
		if q.question != last {
			return q.question, q.expectedAnswer, nil
		}
	}
	return qs[0].question, qs[0].expectedAnswer, nil
}

// transferQuestion is the verification question posed after a correct answer
// (README §2 Stage 4 VERIFY). It is deliberately different from every
// practice question, so the domain's distinctness rule
// (ErrTransferQuestionNotDistinct) never trips.
func transferQuestion(topic string) (question, expectedAnswer string, err error) {
	if topic != "Addition" {
		return "", "", ErrTopicNotYetSupported
	}
	return "What is 5 + 6?", "11", nil
}

// explanation is the plain-language explanation served to a learner who is
// uncertain, before a fresh question is asked (README §9).
func explanation(topic string) string {
	if topic == "Addition" {
		return "Addition means putting two groups together and counting all the items to find the total."
	}
	return ""
}

// completionMessage is what the learner is told when a concept is verified
// mastered (README §2, LOOP OR CLOSE: briefly confirm mastery and stop).
func completionMessage(topic string) string {
	return "That's right! You have mastered " + topic + "."
}

// diagnosticPrompt is the server-owned opening question of the DIAGNOSE stage
// (README §2 Stage 1) for a topic.
func diagnosticPrompt(topic string) (string, error) {
	switch topic {
	case "Addition":
		return "What do you already know about addition?", nil
	default:
		return "", ErrTopicNotYetSupported
	}
}
