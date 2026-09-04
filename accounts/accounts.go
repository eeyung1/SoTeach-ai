// Package accounts implements the guardian account + verifiable consent layer
// that README §10 and workingReadme.md §6 require before a child's data is
// collected or any money can involve a minor (Nigeria Data Protection Act
// 2023). Consent is recorded as real account-flow state: who consented, for
// which learner, when, to what text — not a database boolean.
package accounts

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Errors the accounts layer reports. Consumers map them to HTTP statuses.
var (
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("an account with that email already exists")
	ErrEmailRequired      = errors.New("email is required")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrLearnerRequired    = errors.New("learner name is required")
)

// minPasswordLength is the shortest acceptable guardian password.
const minPasswordLength = 8

// consentTextVersion is the exact consent wording a guardian accepts. Keeping
// it versioned means a recorded consent can always be shown verbatim.
const consentText = "I, the parent or guardian of the learner named in this record, consent to SoTeach collecting and using that learner's learning data (diagnostic findings and progress) solely to provide tutoring, in line with the Nigeria Data Protection Act 2023."

// Guardian is a parent/guardian account.
type Guardian struct {
	Email        string    `json:"email"`
	PasswordHash string    `json:"passwordHash"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Consent is one guardian's recorded, verifiable consent for one learner.
type Consent struct {
	GuardianEmail string    `json:"guardianEmail"`
	Learner       string    `json:"learner"`
	ConsentText   string    `json:"consentText"`
	ConsentedAt   time.Time `json:"consentedAt"`
	Token         string    `json:"token"`
}

// Service is the application boundary for guardian accounts and consent. The
// tutoring engine never depends on it; it exists alongside, gating real data
// collection and the future paid funnel.
type Service struct {
	store Store
}

// NewService creates the account service over the given store.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Register creates a guardian account. The email is normalized (lower-cased,
// trimmed) and must be unique; the password is bcrypt-hashed and never stored
// in plain text.
func (s *Service) Register(email, password string) (Guardian, error) {
	email = normalizeEmail(email)
	if email == "" {
		return Guardian{}, ErrEmailRequired
	}
	if len(password) < minPasswordLength {
		return Guardian{}, ErrPasswordTooShort
	}
	if _, err := s.store.GuardianByEmail(email); err == nil {
		return Guardian{}, ErrEmailTaken
	} else if !errors.Is(err, ErrNotFound) {
		return Guardian{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Guardian{}, err
	}
	g := Guardian{Email: email, PasswordHash: string(hash), CreatedAt: time.Now().UTC()}
	if err := s.store.SaveGuardian(g); err != nil {
		return Guardian{}, err
	}
	return g, nil
}

// Login verifies a guardian's password and issues a session token.
func (s *Service) Login(email, password string) (string, error) {
	email = normalizeEmail(email)
	g, err := s.store.GuardianByEmail(email)
	if errors.Is(err, ErrNotFound) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(g.PasswordHash), []byte(password)) != nil {
		return "", ErrInvalidCredentials
	}

	token, err := randomToken()
	if err != nil {
		return "", err
	}
	if err := s.store.SaveSessionToken(token, g.Email); err != nil {
		return "", err
	}
	return token, nil
}

// Authenticate resolves a session token to the guardian's email, or
// ErrUnauthorized.
func (s *Service) Authenticate(token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", ErrUnauthorized
	}
	email, err := s.store.EmailByToken(token)
	if errors.Is(err, ErrNotFound) {
		return "", ErrUnauthorized
	}
	if err != nil {
		return "", err
	}
	return email, nil
}

// Consent records an explicit, verifiable consent by guardianEmail for the
// learner. A later consent for the same learner replaces the earlier record.
func (s *Service) Consent(guardianEmail, learner string) (Consent, error) {
	guardianEmail = normalizeEmail(guardianEmail)
	learner = strings.TrimSpace(learner)
	if guardianEmail == "" {
		return Consent{}, ErrUnauthorized
	}
	if learner == "" {
		return Consent{}, ErrLearnerRequired
	}

	token, err := randomToken()
	if err != nil {
		return Consent{}, err
	}
	c := Consent{
		GuardianEmail: guardianEmail,
		Learner:       learner,
		ConsentText:   consentText,
		ConsentedAt:   time.Now().UTC(),
		Token:         token,
	}
	if err := s.store.SaveConsent(c); err != nil {
		return Consent{}, err
	}
	return c, nil
}

// ConsentFor returns the current consent record for a learner, if any.
func (s *Service) ConsentFor(learner string) (Consent, bool) {
	c, err := s.store.ConsentByLearner(strings.TrimSpace(learner))
	if err != nil {
		return Consent{}, false
	}
	return c, true
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
