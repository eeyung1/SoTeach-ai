package session

import "errors"

// ErrLearnerNotActive is returned when an answer is submitted on behalf
// of a learner who is not the currently active respondent (README §3,
// rule "One active respondent"; Blueprint lesson #1, "explicit turn
// management").
var ErrLearnerNotActive = errors.New("this learner is not the active respondent")

// ErrLearnerNotFound is returned when an answer is submitted for a name
// that is not part of the roster at all.
var ErrLearnerNotFound = errors.New("learner not found in this session")

// Roster tracks multiple learners sharing one tutoring context and knows
// whose turn it is. Each learner still owns their own independent
// Session (README §3, rule "Multiple learners, separate state") —
// Roster only adds turn ownership on top; it does not duplicate or
// merge any learner's underlying state.
type Roster struct {
	sessions map[string]*Session
	order    []string
	active   string
}

// NewRoster creates a multi-learner context. The first learner listed
// becomes the initially active respondent.
func NewRoster(learnerNames ...string) *Roster {
	r := &Roster{
		sessions: make(map[string]*Session, len(learnerNames)),
		order:    learnerNames,
	}
	for _, name := range learnerNames {
		r.sessions[name] = New(name)
	}
	if len(learnerNames) > 0 {
		r.active = learnerNames[0]
	}
	return r
}

// Session returns the individual Session belonging to the named learner,
// or nil if that learner is not part of this roster.
func (r *Roster) Session(name string) *Session {
	return r.sessions[name]
}

// ActiveLearner returns the name of the learner whose turn it currently
// is.
func (r *Roster) ActiveLearner() string {
	return r.active
}

// AdvanceTurn moves the active respondent to the next learner in roster
// order (README §9, MOVE_TO_NEXT_LEARNER_OR_QUESTION), wrapping around to
// the first learner after the last. On a roster with zero or one
// learners, it is a no-op — there is no one else for the turn to move to.
func (r *Roster) AdvanceTurn() {
	if len(r.order) < 2 {
		return
	}

	for i, name := range r.order {
		if name == r.active {
			r.active = r.order[(i+1)%len(r.order)]
			return
		}
	}
}

// SubmitAnswerFor submits an answer on behalf of the named learner. It is
// rejected if the learner is not part of the roster (ErrLearnerNotFound)
// or is not the currently active respondent (ErrLearnerNotActive) — a
// learner's response must never accidentally be attributed to another
// learner (README §3). On rejection, no session's state is changed.
func (r *Roster) SubmitAnswerFor(name, answer string) error {
	if _, ok := r.sessions[name]; !ok {
		return ErrLearnerNotFound
	}

	if name != r.active {
		return ErrLearnerNotActive
	}

	r.sessions[name].SubmitAnswer(answer)
	return nil
}
