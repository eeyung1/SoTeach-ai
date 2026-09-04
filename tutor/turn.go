package tutor

import (
	"errors"
	"strings"

	"soteach/ai"
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

// ErrNoDiagnosisYet is returned when a diagnosis report is requested before
// any diagnosis has been completed for the learner.
var ErrNoDiagnosisYet = errors.New("no diagnosis recorded yet for this learner")

// wrongThreshold is how many consecutive incorrect practice answers happen
// before the tutor stops merely re-asking and teaches the gap instead
// (README §9 — no child left floundering in a loop).
const wrongThreshold = 2

// Tutor is the application layer that owns the tutoring loop (README §2;
// Agent.md §11): it decides what happens next in a learner's session and what
// the learner is asked or told. A thin client never sequences the loop — it
// sends the learner's input and receives the next prompt.
//
// Question and explanation content is served by the Tutor itself (a
// deterministic stub for now, to be replaced by AI-generated content when a
// concrete requirement exists — Agent.md §38).
type Tutor struct {
	store    session.Store
	provider ai.AIProvider
}

// NewTutor creates a Tutor over the given session store, with no AI provider
// (diagnosis records the learner's explanation verbatim).
func NewTutor(store session.Store) *Tutor {
	return &Tutor{store: store}
}

// NewAITutor creates a Tutor that evaluates diagnosis through provider
// (Agent.md §38-40): SubmitDiagnosis runs provider.Diagnose on the learner's
// explanation and stores the validated DiagnosticResult. A nil provider keeps
// the verbatim behavior of NewTutor.
func NewAITutor(store session.Store, provider ai.AIProvider) *Tutor {
	return &Tutor{store: store, provider: provider}
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

	question, expected, err := firstPracticeQuestion(s.Topic(), s.GradeBand())
	if err != nil {
		return Outcome{}, err
	}

	if t.provider != nil {
		result, err := t.provider.Diagnose(s.Topic(), explanation)
		if err != nil {
			return Outcome{}, err
		}
		if err := s.RecordDiagnosticResult(result); err != nil {
			return Outcome{}, err
		}
	}
	s.RecordDiagnosticResponse(explanation)
	if err := s.AskQuestion(question, expected); err != nil {
		return Outcome{}, err
	}
	if err := t.store.Save(s); err != nil {
		return Outcome{}, err
	}

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
		s.ResetWrongStreak()
		if err := t.store.Save(s); err != nil {
			return Outcome{}, err
		}
		return Outcome{Prompt: completionMessage(s.Topic(), s.GradeBand())}, nil

	case session.AwaitingTransferCheck:
		s.ResetWrongStreak()
		question, expected, err := transferQuestion(s.Topic(), s.GradeBand())
		if err != nil {
			return Outcome{}, err
		}
		if err := s.AskTransferQuestion(question, expected); err != nil {
			return Outcome{}, err
		}
		if err := t.store.Save(s); err != nil {
			return Outcome{}, err
		}
		return Outcome{Prompt: question}, nil

	case session.Incorrect:
		s.NoteIncorrectAttempt()
		question, expected, err := nextPracticeQuestion(s.Topic(), s.GradeBand(), s.CurrentQuestion())
		if err != nil {
			return Outcome{}, err
		}
		if s.WrongStreak() >= wrongThreshold {
			// Repeatedly wrong: teach the gap, then ask again (README §9).
			teach := explanation(s.Topic(), s.GradeBand())
			s.Teach(teach)
			s.ResetWrongStreak()
			if err := s.AskQuestion(question, expected); err != nil {
				return Outcome{}, err
			}
			if err := t.store.Save(s); err != nil {
				return Outcome{}, err
			}
			return Outcome{Prompt: teach + " " + question}, nil
		}
		if err := s.AskQuestion(question, expected); err != nil {
			return Outcome{}, err
		}
		if err := t.store.Save(s); err != nil {
			return Outcome{}, err
		}
		return Outcome{Prompt: question}, nil

	case session.Uncertain:
		s.ResetWrongStreak()
		teach := explanation(s.Topic(), s.GradeBand())
		question, expected, err := nextPracticeQuestion(s.Topic(), s.GradeBand(), s.CurrentQuestion())
		if err != nil {
			return Outcome{}, err
		}
		s.Teach(teach)
		if err := s.AskQuestion(question, expected); err != nil {
			return Outcome{}, err
		}
		if err := t.store.Save(s); err != nil {
			return Outcome{}, err
		}
		return Outcome{Prompt: teach + " " + question}, nil

	default:
		return Outcome{}, ErrUnexpectedAnswerState
	}
}

// BeginTopic starts (or resumes) a learner's session on a topic — the
// server-side entry to the loop (README §6, SELECT_SUBJECT -> SELECT_TOPIC ->
// DIAGNOSE). The grade band is set up-front so even the first diagnostic
// prompt is calibrated to the learner's age band (README §4/§13). A thin
// client never chooses the diagnostic wording; the server selects
// subject/topic and returns the first thing the learner should respond to.
//
// Beginning a topic the learner already has in progress resumes it at the
// correct state rather than repeating diagnosis (README §6): a pending
// question is returned, and already-recorded mastery is reported, not reset.
// An invalid grade band or a topic with no server-side content is rejected
// without persisting a session (README §5).
func (t *Tutor) BeginTopic(learner, subject, topic, gradeBand string) (Outcome, error) {
	s, err := t.store.Load(learner)
	if errors.Is(err, session.ErrSessionNotFound) {
		s = session.New(learner)
	} else if err != nil {
		return Outcome{}, err
	}

	if err := s.SetGradeBand(gradeBand); err != nil {
		return Outcome{}, err
	}

	prompt, err := diagnosticPrompt(topic, gradeBand)
	if err != nil {
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

	if err := t.store.Save(s); err != nil {
		return Outcome{}, err
	}
	return Outcome{Prompt: next}, nil
}

// ApplyInput routes one piece of learner input to the correct tutoring action
// from the session state alone (README §6): a learner mid-diagnosis is giving
// their explanation, while a learner with a question pending is giving an
// answer. Keeping this routing inside the engine means HTTP handlers never
// decide what an input means (Agent.md §19). It is rejected when no session
// exists for the learner, or when the session is not waiting for any input.
func (t *Tutor) ApplyInput(learner, input string) (Outcome, error) {
	if isStopRequest(input) {
		return t.Stop(learner)
	}

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

const stopPrompt = "Okay, let's stop here. You can begin again any time."

// Stop is the learner's exit affordance: it abandons the current attempt on
// this topic (README §3, handle interruptions; no learner is trapped in a
// loop). The in-progress diagnosis/question state is cleared via
// Session.ResetAttempt, so beginning the same topic again starts fresh, while
// learner-level state (grade band) and any demonstrated mastery are kept.
func (t *Tutor) Stop(learner string) (Outcome, error) {
	s, err := t.store.Load(learner)
	if err != nil {
		return Outcome{}, err
	}
	if s.IsMastered() {
		return Outcome{Prompt: "You have already finished this topic."}, nil
	}

	s.ResetAttempt()
	if err := t.store.Save(s); err != nil {
		return Outcome{}, err
	}
	return Outcome{Prompt: stopPrompt}, nil
}

// stopWords are the plain-language ways a learner can stop, recognized before
// any state routing so a stop word is never treated as a math answer.
var stopWords = map[string]bool{
	"stop":   true,
	"quit":   true,
	"exit":   true,
	"enough": true,
}

func isStopRequest(input string) bool {
	return stopWords[strings.ToLower(strings.TrimSpace(input))]
}

// SetGradeBand records the learner's grade/age band on their saved session
// (README §4/§10) — learner-level state used for calibration. A band outside
// the defined set is rejected via session.ErrInvalidGradeBand and leaves the
// session unchanged. The learner must already have a begun session.
func (t *Tutor) SetGradeBand(learner, band string) error {
	s, err := t.store.Load(learner)
	if err != nil {
		return err
	}
	if err := s.SetGradeBand(band); err != nil {
		return err
	}
	return t.store.Save(s)
}

// DiagnosisReport is the learner/parent-facing summary of what diagnosis
// found — the free "value moment" of the acquisition funnel (Blueprint/
// monetization_funnel.md): the learner's own words plus, when an AI evaluation
// was recorded, the structured gap. Evaluation fields are empty when no
// provider was used.
type DiagnosisReport struct {
	Learner           string   `json:"learner"`
	GradeBand         string   `json:"gradeBand"`
	Subject           string   `json:"subject"`
	Topic             string   `json:"topic"`
	Finding           string   `json:"finding"`
	Concept           string   `json:"concept,omitempty"`
	KnownSkills       []string `json:"knownSkills,omitempty"`
	Gaps              []string `json:"gaps,omitempty"`
	Misconceptions    []string `json:"misconceptions,omitempty"`
	Confidence        float64  `json:"confidence,omitempty"`
	RecommendedAction string   `json:"recommendedAction,omitempty"`
}

// DiagnosticReport returns the diagnosis report for a learner whose diagnosis
// is complete, or ErrNoDiagnosisYet if none has been recorded yet.
func (t *Tutor) DiagnosticReport(learner string) (DiagnosisReport, error) {
	s, err := t.store.Load(learner)
	if err != nil {
		return DiagnosisReport{}, err
	}
	if !s.DiagnosisComplete() {
		return DiagnosisReport{}, ErrNoDiagnosisYet
	}

	report := DiagnosisReport{
		Learner:   s.LearnerName(),
		GradeBand: s.GradeBand(),
		Subject:   s.Subject(),
		Topic:     s.Topic(),
		Finding:   s.DiagnosticFinding(),
	}
	if res := s.LastDiagnosticResult(); res.Concept != "" {
		report.Concept = res.Concept
		report.KnownSkills = res.KnownSkills
		report.Gaps = res.Gaps
		report.Misconceptions = res.Misconceptions
		report.Confidence = res.Confidence
		report.RecommendedAction = res.RecommendedAction
	}
	return report, nil
}

// qa pairs a question with its deterministic expected answer.
type qa struct {
	question       string
	expectedAnswer string
}

// topicContent is the full set of learner-facing content for one topic, in one
// grade band's voice (README §4 / Agent.md §13 — age calibration is a product
// and safety requirement, not cosmetic).
type topicContent struct {
	diagnosticPrompt string
	explanation      string
	completion       string
	practice         []qa
	transfer         qa
}

// defaultBand is the fallback for an empty/unknown grade band. It carries the
// wording that existed before calibration, so existing sessions and tests see
// exactly today's strings.
const defaultBand = "JSS1-3"

// additionByBand holds the Addition content in each of the three grade bands.
// Answers stay deterministic numbers so checking is unchanged; only the voice
// differs.
var additionByBand = map[string]topicContent{
	"Primary 4-6": {
		diagnosticPrompt: "Let's practise adding. First, tell me in your own words: can you already add small numbers, like 2 + 2?",
		explanation:      "Adding means putting two groups together and counting everything you have. If you have 4 sweets and someone gives you 2 more, you count them all: 5, 6 — that is 6 sweets.",
		completion:       "That's right! You have mastered adding numbers.",
		practice: []qa{
			{question: "You have 7 sweets and a friend gives you 3 more. How many sweets do you have now?", expectedAnswer: "10"},
			{question: "There are 4 birds on a tree and 6 more land on it. How many birds are there now?", expectedAnswer: "10"},
			{question: "Ade reads 2 books on Monday and 5 books on Tuesday. How many books did he read altogether?", expectedAnswer: "7"},
		},
		transfer: qa{question: "Mummy bakes 5 buns and then bakes 6 more. How many buns does she bake altogether?", expectedAnswer: "11"},
	},
	"JSS1-3": {
		diagnosticPrompt: "What do you already know about addition?",
		explanation:      "Addition means putting two groups together and counting all the items to find the total.",
		completion:       "That's right! You have mastered Addition.",
		practice: []qa{
			{question: "What is 7 + 3?", expectedAnswer: "10"},
			{question: "What is 4 + 6?", expectedAnswer: "10"},
			{question: "What is 2 + 5?", expectedAnswer: "7"},
		},
		transfer: qa{question: "What is 5 + 6?", expectedAnswer: "11"},
	},
	"SSS1-3": {
		diagnosticPrompt: "Before we proceed, explain in your own words what you understand by the addition operation and the properties you know, such as commutativity.",
		explanation:      "Addition is a binary operation that combines two numbers, called addends, into a single total known as the sum. For example, in 7 + 3 = 10, 7 and 3 are the addends and 10 is the sum. Addition is commutative: 7 + 3 = 3 + 7.",
		completion:       "Correct. You have demonstrated mastery of addition.",
		practice: []qa{
			{question: "Evaluate: 7 + 3.", expectedAnswer: "10"},
			{question: "Find the sum of 4 and 6.", expectedAnswer: "10"},
			{question: "Compute: 2 + 5.", expectedAnswer: "7"},
		},
		transfer: qa{question: "Determine the value of 5 + 6.", expectedAnswer: "11"},
	},
}

// contentFor returns the content bundle for topic in band. Only Addition has
// content yet (README §5); an empty/unknown band falls back to the JSS1-3
// bundle so the pre-calibration wording is preserved.
func contentFor(topic, band string) (topicContent, error) {
	if topic != "Addition" {
		return topicContent{}, ErrTopicNotYetSupported
	}
	if c, ok := additionByBand[band]; ok {
		return c, nil
	}
	return additionByBand[defaultBand], nil
}

// firstPracticeQuestion returns the first practice question for topic/band,
// with its deterministic expected answer.
func firstPracticeQuestion(topic, band string) (question, expectedAnswer string, err error) {
	c, err := contentFor(topic, band)
	if err != nil {
		return "", "", err
	}
	return c.practice[0].question, c.practice[0].expectedAnswer, nil
}

// nextPracticeQuestion returns a practice question different from last for
// topic/band, so an incorrect or uncertain answer loops back to a fresh
// question for the same gap rather than repeating the identical one.
func nextPracticeQuestion(topic, band, last string) (question, expectedAnswer string, err error) {
	c, err := contentFor(topic, band)
	if err != nil {
		return "", "", err
	}
	for _, q := range c.practice {
		if q.question != last {
			return q.question, q.expectedAnswer, nil
		}
	}
	return c.practice[0].question, c.practice[0].expectedAnswer, nil
}

// transferQuestion is the verification question posed after a correct answer
// (README §2 Stage 4 VERIFY). It is deliberately different from every practice
// question, so the domain's distinctness rule (ErrTransferQuestionNotDistinct)
// never trips.
func transferQuestion(topic, band string) (question, expectedAnswer string, err error) {
	c, err := contentFor(topic, band)
	if err != nil {
		return "", "", err
	}
	return c.transfer.question, c.transfer.expectedAnswer, nil
}

// explanation is the plain-language explanation served to a learner who is
// uncertain or repeatedly wrong, before a fresh question is asked (README §9),
// calibrated to the learner's band.
func explanation(topic, band string) string {
	c, err := contentFor(topic, band)
	if err != nil {
		return ""
	}
	return c.explanation
}

// completionMessage is what the learner is told when a concept is verified
// mastered (README §2, LOOP OR CLOSE), in the learner's band's voice.
func completionMessage(topic, band string) string {
	c, err := contentFor(topic, band)
	if err != nil {
		return ""
	}
	return c.completion
}

// diagnosticPrompt is the server-owned opening question of the DIAGNOSE stage
// (README §2 Stage 1) for a topic, calibrated to the learner's band.
func diagnosticPrompt(topic, band string) (string, error) {
	c, err := contentFor(topic, band)
	if err != nil {
		return "", err
	}
	return c.diagnosticPrompt, nil
}

// Subject is one curriculum subject the server can teach, with its topics.
type Subject struct {
	Name   string   `json:"name"`
	Topics []string `json:"topics"`
}

// Curriculum returns the subjects and topics the server currently has content
// for (workingReadme §3: the curriculum is server-owned; clients render it,
// they never decide it). This is the narrow MVP set — one subject, one topic
// (README §5) — and grows as content is added. It must stay in sync with the
// content helpers above.
func Curriculum() []Subject {
	return []Subject{{Name: "Mathematics", Topics: []string{"Addition"}}}
}
