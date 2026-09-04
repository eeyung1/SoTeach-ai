package session

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"soteach/ai"
)

// ErrSessionNotFound is returned by a Store's Load when no session has yet
// been saved for the requested learner.
var ErrSessionNotFound = errors.New("no saved session for this learner")

// Store is the persistence boundary for Sessions (Agent.md §20: prove the
// behavior in memory first, then make it durable). The tutor and API depend
// only on this interface, so swapping the in-memory store for a durable one
// changes nothing in tutoring or HTTP behavior.
type Store interface {
	// Save records the current authoritative state of s under its learner
	// name. It reports an error when the store cannot persist (e.g. a disk
	// write fails), so a failed save is never silently treated as success.
	Save(*Session) error
	// Load returns a fresh Session reconstructed from the state saved for
	// name, or ErrSessionNotFound if none has been saved.
	Load(name string) (*Session, error)
}

// MemoryStore persists Sessions in memory, keyed by learner name, so an
// unfinished session can be resumed within one process (README §6). It
// implements Store for the in-memory proof of the persistence behavior.
//
// Save captures an independent copy of the Session's authoritative state at
// the moment it is called — later mutations to the live Session do not leak
// into what was saved — and Load returns a fresh, fully resumed Session, not
// a shared pointer.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]sessionSnapshot
}

// NewMemoryStore creates an empty in-memory session store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sessions: map[string]sessionSnapshot{}}
}

// Save records the current authoritative state of s under s's learner name.
func (m *MemoryStore) Save(s *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.LearnerName()] = captureSnapshot(s)
	return nil
}

// Load returns a fresh Session reconstructed from the state saved for name,
// or ErrSessionNotFound if no session has been saved for that learner.
func (m *MemoryStore) Load(name string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snap, ok := m.sessions[name]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return restoreSnapshot(snap), nil
}

// FileStore is a durable Store: each learner's session snapshot is written as
// a JSON file under one directory, so a session survives process restarts
// (Agent.md §20, persist it second). It deliberately keeps the same Store
// contract as MemoryStore — the tutor, API, and web client do not change.
type FileStore struct {
	mu  sync.Mutex
	dir string
}

// NewFileStore opens (creating if needed) a durable store rooted at dir.
func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &FileStore{dir: dir}, nil
}

// Save serializes the session's authoritative state to the learner's file.
func (f *FileStore) Save(s *Session) error {
	data, err := json.Marshal(captureSnapshot(s))
	if err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if err := os.WriteFile(f.path(s.LearnerName()), data, 0o600); err != nil {
		return err
	}
	return nil
}

// Load reads and rebuilds the session saved for name, or returns
// ErrSessionNotFound if no file exists for that learner.
func (f *FileStore) Load(name string) (*Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := os.ReadFile(f.path(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	var snap sessionSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	return restoreSnapshot(snap), nil
}

// path maps a learner name to its snapshot file. The name is hex-encoded so
// any single URL path segment (the only kind that can reach the store) is a
// safe filename.
func (f *FileStore) path(name string) string {
	return filepath.Join(f.dir, hex.EncodeToString([]byte(name))+".json")
}

// sessionSnapshot is a plain-value, serializable copy of everything that makes
// up a Session's authoritative state. Keeping it separate from Session means a
// store never holds a live pointer whose fields a caller could mutate
// underneath it, and it can be written to disk as JSON.
type sessionSnapshot struct {
	LearnerName          string             `json:"learnerName"`
	GradeBand            string             `json:"gradeBand"`
	Subject              string             `json:"subject"`
	Topic                string             `json:"topic"`
	State                State              `json:"state"`
	Question             string             `json:"question"`
	ExpectedAnswer       string             `json:"expectedAnswer"`
	Answer               string             `json:"answer"`
	HasAnswer            bool               `json:"hasAnswer"`
	Mastered             bool               `json:"mastered"`
	MasteryBySubject     map[string]bool    `json:"masteryBySubject"`
	ConfirmationCount    int                `json:"confirmationCount"`
	DiagnosticPrompt     string             `json:"diagnosticPrompt"`
	DiagnosticFinding    string             `json:"diagnosticFinding"`
	DiagnosticResult     ai.DiagnosticResult `json:"diagnosticResult"`
	DiagnosisComplete    bool               `json:"diagnosisComplete"`
	OriginalQuestion     string             `json:"originalQuestion"`
	IsTransferQuestion   bool               `json:"isTransferQuestion"`
	TaughtSinceUncertain bool               `json:"taughtSinceUncertain"`
	LastTeaching         string             `json:"lastTeaching"`
	WrongStreak          int                `json:"wrongStreak"`
}

// captureSnapshot copies a Session into a sessionSnapshot. Maps and slices
// are cloned so the snapshot shares no mutable memory with the live Session.
func captureSnapshot(s *Session) sessionSnapshot {
	return sessionSnapshot{
		LearnerName:          s.learnerName,
		GradeBand:            s.gradeBand,
		Subject:              s.subject,
		Topic:                s.topic,
		State:                s.state,
		Question:             s.question,
		ExpectedAnswer:       s.expectedAnswer,
		Answer:               s.answer,
		HasAnswer:            s.hasAnswer,
		Mastered:             s.mastered,
		MasteryBySubject:     cloneBoolMap(s.masteryBySubject),
		ConfirmationCount:    s.confirmationCount,
		DiagnosticPrompt:     s.diagnosticPrompt,
		DiagnosticFinding:    s.diagnosticFinding,
		DiagnosticResult:     cloneDiagnosticResult(s.diagnosticResult),
		DiagnosisComplete:    s.diagnosisComplete,
		OriginalQuestion:     s.originalQuestion,
		IsTransferQuestion:   s.isTransferQuestion,
		TaughtSinceUncertain: s.taughtSinceUncertain,
		LastTeaching:         s.lastTeaching,
		WrongStreak:          s.wrongStreak,
	}
}

// restoreSnapshot builds a fresh Session from a sessionSnapshot, cloning the
// snapshot's maps/slices so the restored Session is likewise independent.
func restoreSnapshot(snap sessionSnapshot) *Session {
	s := &Session{
		learnerName:          snap.LearnerName,
		gradeBand:            snap.GradeBand,
		subject:              snap.Subject,
		topic:                snap.Topic,
		state:                snap.State,
		question:             snap.Question,
		expectedAnswer:       snap.ExpectedAnswer,
		answer:               snap.Answer,
		hasAnswer:            snap.HasAnswer,
		mastered:             snap.Mastered,
		masteryBySubject:     cloneBoolMap(snap.MasteryBySubject),
		confirmationCount:    snap.ConfirmationCount,
		diagnosticPrompt:     snap.DiagnosticPrompt,
		diagnosticFinding:    snap.DiagnosticFinding,
		diagnosticResult:     cloneDiagnosticResult(snap.DiagnosticResult),
		diagnosisComplete:    snap.DiagnosisComplete,
		originalQuestion:     snap.OriginalQuestion,
		isTransferQuestion:   snap.IsTransferQuestion,
		taughtSinceUncertain: snap.TaughtSinceUncertain,
		lastTeaching:         snap.LastTeaching,
		wrongStreak:          snap.WrongStreak,
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
