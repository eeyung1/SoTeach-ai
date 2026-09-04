package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"soteach/accounts"
)

// guardianAPI handles the guardian account + consent endpoints. It is a
// separate surface from the tutoring API: guardians gate real data collection
// and the future paid funnel, but never touch tutoring logic.
type guardianAPI struct {
	svc *accounts.Service
}

// AddGuardianRoutes registers the guardian/consent routes onto mux.
func AddGuardianRoutes(mux *http.ServeMux, svc *accounts.Service) {
	g := &guardianAPI{svc: svc}

	mux.HandleFunc("POST /guardians/register", g.register)
	mux.HandleFunc("POST /guardians/login", g.login)
	mux.HandleFunc("POST /guardians/consent", g.consent)
	mux.HandleFunc("GET /guardians/consent/{learner}", g.consentStatus)
	mux.HandleFunc("GET /learners/{learner}/consent", g.publicConsentStatus)
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type consentRequest struct {
	Learner string `json:"learner"`
}

// register creates a guardian account.
func (g *guardianAPI) register(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}
	if _, err := g.svc.Register(req.Email, req.Password); err != nil {
		writeError(w, guardianStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"email": strings.ToLower(strings.TrimSpace(req.Email))})
}

// login verifies a guardian's password and issues a session token.
func (g *guardianAPI) login(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}
	token, err := g.svc.Login(req.Email, req.Password)
	if err != nil {
		writeError(w, guardianStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"token": token,
		"email": strings.ToLower(strings.TrimSpace(req.Email)),
	})
}

// authenticatedEmail resolves the Bearer token in the request, writing a 401
// and returning "" if it is missing or invalid.
func (g *guardianAPI) authenticatedEmail(w http.ResponseWriter, r *http.Request) string {
	raw := r.Header.Get("Authorization")
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		writeError(w, http.StatusUnauthorized, accounts.ErrUnauthorized.Error())
		return ""
	}
	email, err := g.svc.Authenticate(parts[1])
	if err != nil {
		writeError(w, http.StatusUnauthorized, accounts.ErrUnauthorized.Error())
		return ""
	}
	return email
}

// consent records an authenticated guardian's explicit consent for a learner.
func (g *guardianAPI) consent(w http.ResponseWriter, r *http.Request) {
	email := g.authenticatedEmail(w, r)
	if email == "" {
		return
	}

	var req consentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}
	c, err := g.svc.Consent(email, req.Learner)
	if err != nil {
		writeError(w, guardianStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"consented":   true,
		"learner":     c.Learner,
		"guardian":    c.GuardianEmail,
		"consentedAt": c.ConsentedAt,
	})
}

// consentStatus reports, to the authenticated guardian, the consent status of
// a learner.
func (g *guardianAPI) consentStatus(w http.ResponseWriter, r *http.Request) {
	if g.authenticatedEmail(w, r) == "" {
		return
	}
	g.writeConsentStatus(w, r.PathValue("learner"))
}

// publicConsentStatus lets any client read whether a learner has consent, so
// the UI can gate (consent text is not exposed publicly).
func (g *guardianAPI) publicConsentStatus(w http.ResponseWriter, r *http.Request) {
	g.writeConsentStatus(w, r.PathValue("learner"))
}

func (g *guardianAPI) writeConsentStatus(w http.ResponseWriter, learner string) {
	if learner == "" {
		writeError(w, http.StatusBadRequest, accounts.ErrLearnerRequired.Error())
		return
	}
	resp := map[string]any{"learner": learner, "consented": false, "guardian": ""}
	if c, ok := g.svc.ConsentFor(learner); ok {
		resp["consented"] = true
		resp["guardian"] = c.GuardianEmail
		resp["consentedAt"] = c.ConsentedAt
	}
	writeJSON(w, http.StatusOK, resp)
}

// guardianStatus maps accounts errors to HTTP statuses.
func guardianStatus(err error) int {
	switch {
	case errors.Is(err, accounts.ErrEmailTaken):
		return http.StatusConflict
	case errors.Is(err, accounts.ErrEmailRequired),
		errors.Is(err, accounts.ErrPasswordTooShort),
		errors.Is(err, accounts.ErrLearnerRequired):
		return http.StatusBadRequest
	case errors.Is(err, accounts.ErrInvalidCredentials),
		errors.Is(err, accounts.ErrUnauthorized):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
