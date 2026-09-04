package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"soteach/accounts"
	"soteach/api"
)

// jsonBody marshals v into a request body.
func jsonBody(v any) *bytes.Reader {
	raw, _ := json.Marshal(v)
	return bytes.NewReader(raw)
}

// guardianHandler returns an http.Handler with just the guardian routes.
func guardianHandler() http.Handler {
	mux := http.NewServeMux()
	api.AddGuardianRoutes(mux, accounts.NewService(accounts.NewMemoryStore()))
	return mux
}

// TestGuardianRegisterLoginConsentFlow walks the whole gate over HTTP: register
// a guardian, log in, give consent for a learner, and read it back.
func TestGuardianRegisterLoginConsentFlow(t *testing.T) {
	handler := guardianHandler()

	reg := postJSON(t, handler, http.MethodPost, "/guardians/register", map[string]string{
		"email": "parent@example.com", "password": "password123",
	})
	if reg.Code != http.StatusCreated {
		t.Fatalf("expected 201 on register, got %d", reg.Code)
	}

	// Duplicate email -> 409.
	if rec := postJSON(t, handler, http.MethodPost, "/guardians/register", map[string]string{
		"email": "parent@example.com", "password": "password123",
	}); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate email, got %d", rec.Code)
	}

	// Wrong password -> 401.
	if rec := postJSON(t, handler, http.MethodPost, "/guardians/login", map[string]string{
		"email": "parent@example.com", "password": "nope",
	}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on wrong password, got %d", rec.Code)
	}

	login := postJSON(t, handler, http.MethodPost, "/guardians/login", map[string]string{
		"email": "parent@example.com", "password": "password123",
	})
	if login.Code != http.StatusOK {
		t.Fatalf("expected 200 on login, got %d", login.Code)
	}
	var loginBody struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(login.Body).Decode(&loginBody); err != nil || loginBody.Token == "" {
		t.Fatalf("expected a token from login, got error %v", err)
	}

	// Consent without a token -> 401.
	if rec := postJSON(t, handler, http.MethodPost, "/guardians/consent", map[string]string{
		"learner": "Amaka",
	}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for consent without a token, got %d", rec.Code)
	}

	// Consent with the token -> 200.
	consentReq := httptest.NewRequest(http.MethodPost, "/guardians/consent", jsonBody(map[string]string{"learner": "Amaka"}))
	consentReq.Header.Set("Content-Type", "application/json")
	consentReq.Header.Set("Authorization", "Bearer "+loginBody.Token)
	consentRec := httptest.NewRecorder()
	handler.ServeHTTP(consentRec, consentReq)
	if consentRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on consent, got %d", consentRec.Code)
	}

	// Public status reflects the consent.
	status := httptest.NewRecorder()
	handler.ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/learners/Amaka/consent", nil))
	if status.Code != http.StatusOK {
		t.Fatalf("expected 200 on consent status, got %d", status.Code)
	}
	var statusBody struct {
		Consented bool   `json:"consented"`
		Guardian  string `json:"guardian"`
	}
	if err := json.NewDecoder(status.Body).Decode(&statusBody); err != nil {
		t.Fatalf("expected a JSON body, got error: %v", err)
	}
	if !statusBody.Consented {
		t.Fatal("expected the learner to be consented")
	}
	if statusBody.Guardian != "parent@example.com" {
		t.Fatalf("expected the consenting guardian, got %q", statusBody.Guardian)
	}
}

// TestLearnerWithoutConsentReportsUnconsented covers the default state.
func TestLearnerWithoutConsentReportsUnconsented(t *testing.T) {
	handler := guardianHandler()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/learners/Bello/consent", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body struct {
		Consented bool `json:"consented"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("expected a JSON body, got error: %v", err)
	}
	if body.Consented {
		t.Fatal("expected an unconsented learner to report consented=false")
	}
}
