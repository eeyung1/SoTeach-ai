package session

import (
	"errors"

	"soteach/ai"
)

// ErrSessionNotFound is returned by MemoryStore.Load when no session has yet
// been saved for the requested learner.
var ErrSessionNotFound = errors.New("no saved session for this learner")

// MemoryStore persists Sessions in memory, keyed by learner name, so an
// unfinished session can be resumed (README §6) without yet introducing a
// database (Agent.md §20: prove behavior in memory first, persist it second;
// PostgreSQL is added only when this behavior is proven).
//
// Save captures an independent copy of the Session's authoritative state at
// the moment it is called — later mutations to the live Session do not leak
// into what was saved — and Load returns a fresh, fully resumed Session, not
// a shared pointer.
type MemoryStore struct {
	sessions map[string]sessionSnapshot
}

// NewMemoryStore creates an empty in-memory session store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sessions: map[string]sessionSnapshot{}}
}

// Save records the current authoritative state of s under s's learner name.
func (m *MemoryStore) Save(s *Session) {
	m.sessions[s.LearnerName()] = captureSnapshot(s)
}

// Load returns a fresh Session reconstructed from the state saved for name,
// or ErrSessionNotFound if no session has been saved for that learner.
func (m *MemoryStore) Load(name string) (*Session, error) {
	snap, ok := m.sessions[name]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return restoreSnapshot(snap), nil
}

// sessionSnapshot is a plain-value copy of everything that makes up a
// Session's authoritative state. Keeping it separate from Session means the
// store never holds a live pointer whose fields a caller could mutate
// underneath it.
type sessionSnapshot struct {
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
}

// captureSnapshot copies a Session into a sessionSnapshot. Maps and slices
// are cloned so the snapshot shares no mutable memory with the live Session.
func captureSnapshot(s *Session) sessionSnapshot {
	return sessionSnapshot{
		learnerName:          s.learnerName,
		gradeBand:            s.gradeBand,
		subject:              s.subject,
		topic:                s.topic,
		state:                s.state,
		question:             s.question,
		expectedAnswer:       s.expectedAnswer,
		answer:               s.answer,
		hasAnswer:            s.hasAnswer,
		mastered:             s.mastered,
		masteryBySubject:     cloneBoolMap(s.masteryBySubject),
		confirmationCount:    s.confirmationCount,
		diagnosticPrompt:     s.diagnosticPrompt,
		diagnosticFinding:    s.diagnosticFinding,
		diagnosticResult:     cloneDiagnosticResult(s.diagnosticResult),
		diagnosisComplete:    s.diagnosisComplete,
		originalQuestion:     s.originalQuestion,
		isTransferQuestion:   s.isTransferQuestion,
		taughtSinceUncertain: s.taughtSinceUncertain,
		lastTeaching:         s.lastTeaching,
	}
}

// restoreSnapshot builds a fresh Session from a sessionSnapshot, cloning the
// snapshot's maps/slices so the restored Session is likewise independent.
func restoreSnapshot(snap sessionSnapshot) *Session {
	s := &Session{
		learnerName:          snap.learnerName,
		gradeBand:            snap.gradeBand,
		subject:              snap.subject,
		topic:                snap.topic,
		state:                snap.state,
		question:             snap.question,
		expectedAnswer:       snap.expectedAnswer,
		answer:               snap.answer,
		hasAnswer:            snap.hasAnswer,
		mastered:             snap.mastered,
		masteryBySubject:     cloneBoolMap(snap.masteryBySubject),
		confirmationCount:    snap.confirmationCount,
		diagnosticPrompt:     snap.diagnosticPrompt,
		diagnosticFinding:    snap.diagnosticFinding,
		diagnosticResult:     cloneDiagnosticResult(snap.diagnosticResult),
		diagnosisComplete:    snap.diagnosisComplete,
		originalQuestion:     snap.originalQuestion,
		isTransferQuestion:   snap.isTransferQuestion,
		taughtSinceUncertain: snap.taughtSinceUncertain,
		lastTeaching:         snap.lastTeaching,
	}
	return s
}

func cloneBoolMap(m map[string]bool) map[string]bool {
	out := make(map[string]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func cloneDiagnosticResult(r ai.DiagnosticResult) ai.DiagnosticResult {
	out := r
	out.KnownSkills = append([]string(nil), r.KnownSkills...)
	out.Gaps = append([]string(nil), r.Gaps...)
	out.Misconceptions = append([]string(nil), r.Misconceptions...)
	return out
}
