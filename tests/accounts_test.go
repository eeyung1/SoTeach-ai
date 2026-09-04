package tests

import (
	"errors"
	"strings"
	"testing"

	"soteach/accounts"
)

func newTestAccounts() *accounts.Service {
	return accounts.NewService(accounts.NewMemoryStore())
}

// TestRegisterCreatesUniqueGuardian covers signup: success, duplicate email,
// too-short password, and missing email.
func TestRegisterCreatesUniqueGuardian(t *testing.T) {
	svc := newTestAccounts()

	if _, err := svc.Register("Guardian@Example.com ", "password123"); err != nil {
		t.Fatalf("expected registration to succeed, got error: %v", err)
	}

	// Duplicate (email is normalized).
	if _, err := svc.Register("guardian@example.com", "password123"); err == nil {
		t.Fatal("expected a duplicate email to be rejected")
	} else if !errors.Is(err, accounts.ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}

	if _, err := svc.Register("new@example.com", "short"); err == nil {
		t.Fatal("expected a too-short password to be rejected")
	} else if !errors.Is(err, accounts.ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}

	if _, err := svc.Register("  ", "password123"); err == nil {
		t.Fatal("expected a missing email to be rejected")
	} else if !errors.Is(err, accounts.ErrEmailRequired) {
		t.Fatalf("expected ErrEmailRequired, got %v", err)
	}
}

// TestLoginIssuesWorkingToken covers login and token authentication.
func TestLoginIssuesWorkingToken(t *testing.T) {
	svc := newTestAccounts()
	if _, err := svc.Register("guardian@example.com", "password123"); err != nil {
		t.Fatalf("expected registration to succeed, got error: %v", err)
	}

	if _, err := svc.Login("guardian@example.com", "wrong-password"); err == nil {
		t.Fatal("expected a wrong password to be rejected")
	} else if !errors.Is(err, accounts.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}

	token, err := svc.Login("guardian@example.com", "password123")
	if err != nil {
		t.Fatalf("expected login to succeed, got error: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty token")
	}

	email, err := svc.Authenticate(token)
	if err != nil {
		t.Fatalf("expected the token to authenticate, got error: %v", err)
	}
	if email != "guardian@example.com" {
		t.Fatalf("expected the guardian's email, got %q", email)
	}

	if _, err := svc.Authenticate("not-a-real-token"); err == nil {
		t.Fatal("expected a bad token to be rejected")
	} else if !errors.Is(err, accounts.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

// TestConsentRecordsVerifiableFields proves consent is a real, auditable
// record — who, which learner, when, to what text, with a token — not a
// boolean.
func TestConsentRecordsVerifiableFields(t *testing.T) {
	svc := newTestAccounts()

	c, err := svc.Consent("guardian@example.com", "Amaka")
	if err != nil {
		t.Fatalf("expected consent to be recorded, got error: %v", err)
	}
	if c.GuardianEmail != "guardian@example.com" {
		t.Fatalf("expected the guardian email to be recorded, got %q", c.GuardianEmail)
	}
	if c.Learner != "Amaka" {
		t.Fatalf("expected the learner to be recorded, got %q", c.Learner)
	}
	if !strings.Contains(c.ConsentText, "Nigeria Data Protection Act") {
		t.Fatal("expected the versioned consent text to be recorded")
	}
	if c.ConsentedAt.IsZero() {
		t.Fatal("expected a consent timestamp to be recorded")
	}
	if c.Token == "" {
		t.Fatal("expected a consent token to be recorded")
	}

	got, ok := svc.ConsentFor("Amaka")
	if !ok {
		t.Fatal("expected consent to be retrievable for the learner")
	}
	if got.Learner != "Amaka" || got.GuardianEmail != "guardian@example.com" {
		t.Fatalf("expected the recorded consent back, got %+v", got)
	}

	if _, ok := svc.ConsentFor("Bello"); ok {
		t.Fatal("expected no consent for an unconsented learner")
	}

	// A later consent for the same learner replaces the earlier record.
	if _, err := svc.Consent("other@example.com", "Amaka"); err != nil {
		t.Fatalf("expected re-consent to be recorded, got error: %v", err)
	}
	got, _ = svc.ConsentFor("Amaka")
	if got.GuardianEmail != "other@example.com" {
		t.Fatalf("expected the latest consent to win, got %q", got.GuardianEmail)
	}
}

// TestSubscribeRecordsPlanAndRejectsUnknown covers the plan/subscription path:
// a valid plan is recorded (id, status, timestamp) and survives a read back; an
// unknown plan or missing account is rejected and leaves nothing recorded.
func TestSubscribeRecordsPlanAndRejectsUnknown(t *testing.T) {
	svc := newTestAccounts()
	if _, err := svc.Register("parent@example.com", "password123"); err != nil {
		t.Fatalf("expected registration to succeed, got error: %v", err)
	}

	if _, ok := svc.Subscription("parent@example.com"); ok {
		t.Fatal("expected no plan before subscribing")
	}

	sub, err := svc.Subscribe("parent@example.com", "family-monthly")
	if err != nil {
		t.Fatalf("expected subscribe to succeed, got error: %v", err)
	}
	if sub.Plan.ID != "family-monthly" {
		t.Fatalf("expected the monthly plan to be recorded, got %q", sub.Plan.ID)
	}
	if sub.Plan.PriceNaira <= 0 {
		t.Fatalf("expected a sample price on the paid plan, got %d", sub.Plan.PriceNaira)
	}
	if sub.Status != "active" {
		t.Fatalf("expected the mock checkout to mark the plan active, got %q", sub.Status)
	}
	if sub.StartedAt.IsZero() {
		t.Fatal("expected a plan start timestamp to be recorded")
	}

	got, ok := svc.Subscription("parent@example.com")
	if !ok {
		t.Fatal("expected the subscription to read back")
	}
	if got.Plan.ID != "family-monthly" || got.Status != "active" {
		t.Fatalf("expected the recorded plan back, got %+v", got)
	}

	if _, err := svc.Subscribe("parent@example.com", "enterprise"); err == nil {
		t.Fatal("expected an unknown plan to be rejected")
	} else if !errors.Is(err, accounts.ErrUnknownPlan) {
		t.Fatalf("expected ErrUnknownPlan, got %v", err)
	}

	if _, err := svc.Subscribe("nobody@example.com", "family-monthly"); err == nil {
		t.Fatal("expected a missing account to be rejected")
	} else if !errors.Is(err, accounts.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a missing account, got %v", err)
	}
}

// TestLearnersListsOnlyOwnGuardiansConsents proves a guardian's Learners view
// (what their subscription covers) contains only the children they personally
// gave consent for, sorted by learner name.
func TestLearnersListsOnlyOwnGuardiansConsents(t *testing.T) {
	svc := newTestAccounts()
	for _, email := range []string{"g1@example.com", "g2@example.com"} {
		if _, err := svc.Register(email, "password123"); err != nil {
			t.Fatalf("expected registration to succeed, got error: %v", err)
		}
	}

	if _, err := svc.Consent("g1@example.com", "Bello"); err != nil {
		t.Fatalf("expected consent to be recorded, got error: %v", err)
	}
	if _, err := svc.Consent("g1@example.com", "Amaka"); err != nil {
		t.Fatalf("expected consent to be recorded, got error: %v", err)
	}
	if _, err := svc.Consent("g2@example.com", "Chidi"); err != nil {
		t.Fatalf("expected consent to be recorded, got error: %v", err)
	}

	one, err := svc.Learners("g1@example.com")
	if err != nil {
		t.Fatalf("expected Learners to succeed, got error: %v", err)
	}
	names := make([]string, len(one))
	for i, c := range one {
		names[i] = c.Learner
	}
	if !equalStrings(names, []string{"Amaka", "Bello"}) {
		t.Fatalf("expected g1's own consented children sorted, got %v", names)
	}

	two, err := svc.Learners("g2@example.com")
	if err != nil {
		t.Fatalf("expected Learners to succeed, got error: %v", err)
	}
	if len(two) != 1 || two[0].Learner != "Chidi" {
		t.Fatalf("expected g2 to see only Chidi, got %+v", two)
	}
}
