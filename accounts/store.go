package accounts

import (
	"sort"
	"sync"
)

// Store is the persistence boundary for the accounts layer. Tutors never touch
// it; guardians and consents are separate from session state.
type Store interface {
	SaveGuardian(g Guardian) error
	GuardianByEmail(email string) (Guardian, error)
	SaveConsent(c Consent) error
	ConsentByLearner(learner string) (Consent, error)
	ConsentsByGuardian(email string) ([]Consent, error)
	SaveSessionToken(token, email string) error
	EmailByToken(token string) (string, error)
}

// MemoryStore is an in-memory Store for tests and the no-persistence path.
type MemoryStore struct {
	mu        sync.RWMutex
	guardians map[string]Guardian // email -> guardian
	consents  map[string]Consent  // learner -> consent
	tokens    map[string]string   // token -> email
}

// NewMemoryStore creates an empty in-memory accounts store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		guardians: map[string]Guardian{},
		consents:  map[string]Consent{},
		tokens:    map[string]string{},
	}
}

func (m *MemoryStore) SaveGuardian(g Guardian) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.guardians[g.Email] = g
	return nil
}

func (m *MemoryStore) GuardianByEmail(email string) (Guardian, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.guardians[email]
	if !ok {
		return Guardian{}, ErrNotFound
	}
	return g, nil
}

func (m *MemoryStore) SaveConsent(c Consent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.consents[c.Learner] = c
	return nil
}

func (m *MemoryStore) ConsentByLearner(learner string) (Consent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.consents[learner]
	if !ok {
		return Consent{}, ErrNotFound
	}
	return c, nil
}

func (m *MemoryStore) ConsentsByGuardian(email string) ([]Consent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Consent
	for _, c := range m.consents {
		if c.GuardianEmail == email {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Learner < out[j].Learner })
	return out, nil
}

func (m *MemoryStore) SaveSessionToken(token, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[token] = email
	return nil
}

func (m *MemoryStore) EmailByToken(token string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	email, ok := m.tokens[token]
	if !ok {
		return "", ErrNotFound
	}
	return email, nil
}

// compile-time conformance.
var _ Store = (*MemoryStore)(nil)
