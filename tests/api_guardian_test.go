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

// guardianLoginToken registers no one; it logs in an existing guardian and
// returns the bearer token.
func guardianLoginToken(t *testing.T, handler http.Handler, email, password string) string {
	t.Helper()
	rec := postJSON(t, handler, http.MethodPost, "/guardians/login", map[string]string{
		"email": email, "password": password,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on login, got %d", rec.Code)
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil || body.Token == "" {
		t.Fatalf("expected a token from login, got error %v", err)
	}
	return body.Token
}

// TestGuardianPlansRequireAuthAndReflectSelection covers the paid-funnel plan
// endpoints (Blueprint/monetization_funnel conversion): every plan endpoint is
// guardian-only; an authenticated guardian can read the sample catalog, choose
// a plan (mock checkout), and /guardians/me reports the active plan alongside
// the consented children that plan covers.
func TestGuardianPlansRequireAuthAndReflectSelection(t *testing.T) {
	handler := guardianHandler()

	// Reject unauthenticated callers.
	for _, p := range []string{"/guardians/plans", "/guardians/me"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for %s without a token, got %d", p, rec.Code)
		}
	}
	if rec := postJSON(t, handler, http.MethodPost, "/guardians/plan", map[string]string{"planID": "family-monthly"}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 choosing a plan without a token, got %d", rec.Code)
	}

	if rec := postJSON(t, handler, http.MethodPost, "/guardians/register", map[string]string{
		"email": "parent@example.com", "password": "password123",
	}); rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on register, got %d", rec.Code)
	}
	token := guardianLoginToken(t, handler, "parent@example.com", "password123")

	authed := func(method, path string, body any) *httptest.ResponseRecorder {
		t.Helper()
		var req *http.Request
		if body == nil {
			req = httptest.NewRequest(method, path, nil)
		} else {
			req = httptest.NewRequest(method, path, jsonBody(body))
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	// Sample catalog is served, with sample prices.
	catalog := authed(http.MethodGet, "/guardians/plans", nil)
	if catalog.Code != http.StatusOK {
		t.Fatalf("expected 200 on plans, got %d", catalog.Code)
	}
	var catalogBody struct {
		Plans []struct {
			ID         string `json:"id"`
			PriceNaira int    `json:"priceNaira"`
		} `json:"plans"`
	}
	if err := json.NewDecoder(catalog.Body).Decode(&catalogBody); err != nil {
		t.Fatalf("expected a JSON body, got error: %v", err)
	}
	found := map[string]bool{}
	for _, p := range catalogBody.Plans {
		found[p.ID] = true
		if p.PriceNaira <= 0 {
			t.Fatalf("expected a sample price on %q, got %d", p.ID, p.PriceNaira)
		}
	}
	if !found["family-monthly"] || !found["family-annual"] {
		t.Fatalf("expected the monthly and annual plans, got %v", catalogBody.Plans)
	}

	// /guardians/me starts with no plan.
	type meBody struct {
		Plan *struct {
			Status string `json:"status"`
			Plan   struct {
				ID string `json:"id"`
			} `json:"plan"`
		} `json:"plan"`
		Learners []struct {
			Learner string `json:"learner"`
		} `json:"learners"`
	}
	var before meBody
	if err := json.NewDecoder(authed(http.MethodGet, "/guardians/me", nil).Body).Decode(&before); err != nil {
		t.Fatalf("expected a JSON body from /guardians/me, got error: %v", err)
	}
	if before.Plan != nil {
		t.Fatal("expected no plan before choosing one")
	}

	// Unknown plan -> 400.
	if rec := authed(http.MethodPost, "/guardians/plan", map[string]string{"planID": "enterprise"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an unknown plan, got %d", rec.Code)
	}

	// Consent for a child, then choose a plan.
	consent := authed(http.MethodPost, "/guardians/consent", map[string]string{"learner": "Amaka"})
	if consent.Code != http.StatusOK {
		t.Fatalf("expected 200 on consent, got %d", consent.Code)
	}
	chosen := authed(http.MethodPost, "/guardians/plan", map[string]string{"planID": "family-monthly"})
	if chosen.Code != http.StatusOK {
		t.Fatalf("expected 200 choosing a plan, got %d", chosen.Code)
	}

	// /guardians/me now reflects the active plan and the consented child.
	var after meBody
	if err := json.NewDecoder(authed(http.MethodGet, "/guardians/me", nil).Body).Decode(&after); err != nil {
		t.Fatalf("expected a JSON body from /guardians/me, got error: %v", err)
	}
	if after.Plan == nil || after.Plan.Plan.ID != "family-monthly" || after.Plan.Status != "active" {
		t.Fatalf("expected the active monthly plan, got %+v", after.Plan)
	}
	if len(after.Learners) != 1 || after.Learners[0].Learner != "Amaka" {
		t.Fatalf("expected the consented child under the guardian, got %+v", after.Learners)
	}
}
