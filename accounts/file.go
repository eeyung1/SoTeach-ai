package accounts

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// fileData is the on-disk shape of the file-backed accounts store.
type fileData struct {
	Guardians map[string]Guardian `json:"guardians"`
	Consents  map[string]Consent  `json:"consents"`
	Sessions  map[string]string   `json:"sessions"`
}

// FileStore persists guardians, consents, and session tokens as one JSON file,
// so accounts survive a server restart without needing a database (mirrors the
// session FileStore philosophy in session/store.go).
type FileStore struct {
	mu   sync.Mutex
	path string
	data fileData
}

// NewFileStore opens (creating if needed) an accounts store rooted at path.
func NewFileStore(path string) (*FileStore, error) {
	fs := &FileStore{path: path}
	fs.data.Guardians = map[string]Guardian{}
	fs.data.Consents = map[string]Consent{}
	fs.data.Sessions = map[string]string{}

	raw, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(raw, &fs.data); err != nil {
			return nil, err
		}
		if fs.data.Guardians == nil {
			fs.data.Guardians = map[string]Guardian{}
		}
		if fs.data.Consents == nil {
			fs.data.Consents = map[string]Consent{}
		}
		if fs.data.Sessions == nil {
			fs.data.Sessions = map[string]string{}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return fs, nil
}

func (f *FileStore) persistLocked() error {
	if dir := filepath.Dir(f.path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	raw, err := json.Marshal(f.data)
	if err != nil {
		return err
	}
	return os.WriteFile(f.path, raw, 0o600)
}

func (f *FileStore) SaveGuardian(g Guardian) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data.Guardians[g.Email] = g
	return f.persistLocked()
}

func (f *FileStore) GuardianByEmail(email string) (Guardian, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	g, ok := f.data.Guardians[email]
	if !ok {
		return Guardian{}, ErrNotFound
	}
	return g, nil
}

func (f *FileStore) SaveConsent(c Consent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data.Consents[c.Learner] = c
	return f.persistLocked()
}

func (f *FileStore) ConsentByLearner(learner string) (Consent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.data.Consents[learner]
	if !ok {
		return Consent{}, ErrNotFound
	}
	return c, nil
}

func (f *FileStore) ConsentsByGuardian(email string) ([]Consent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Consent
	for _, c := range f.data.Consents {
		if c.GuardianEmail == email {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Learner < out[j].Learner })
	return out, nil
}

func (f *FileStore) SaveSessionToken(token, email string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data.Sessions[token] = email
	return f.persistLocked()
}

func (f *FileStore) EmailByToken(token string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	email, ok := f.data.Sessions[token]
	if !ok {
		return "", ErrNotFound
	}
	return email, nil
}

// compile-time conformance.
var _ Store = (*FileStore)(nil)
