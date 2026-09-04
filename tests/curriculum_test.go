package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"soteach/api"
	"soteach/session"
)

// TestCurriculumEndpointListsServerOwnedCatalog proves the server tells the
// client what it can teach (workingReadme §3: clients are render/input layers
// only; the curriculum is server-owned). GET /curriculum returns the available
// subjects with their topics and the defined grade bands — today the narrow
// MVP set (Mathematics → Addition; Primary 4-6, JSS1-3, SSS1-3).
func TestCurriculumEndpointListsServerOwnedCatalog(t *testing.T) {
	handler := api.NewHandler(session.NewMemoryStore())

	req := httptest.NewRequest(http.MethodGet, "/curriculum", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var got struct {
		Subjects []struct {
			Name   string   `json:"name"`
			Topics []string `json:"topics"`
		} `json:"subjects"`
		GradeBands []string `json:"gradeBands"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("expected a JSON body, got error: %v", err)
	}

	if len(got.Subjects) != 1 {
		t.Fatalf("expected one subject, got %d", len(got.Subjects))
	}
	if got.Subjects[0].Name != "Mathematics" {
		t.Fatalf("expected subject Mathematics, got %q", got.Subjects[0].Name)
	}
	foundAddition := false
	for _, topic := range got.Subjects[0].Topics {
		if topic == "Addition" {
			foundAddition = true
		}
	}
	if !foundAddition {
		t.Fatalf("expected topic Addition under Mathematics, got %v", got.Subjects[0].Topics)
	}

	if len(got.GradeBands) != 3 {
		t.Fatalf("expected three grade bands, got %v", got.GradeBands)
	}
}

// TestBeginStoresGradeBandOnResume confirms grade band captured at begin is
// authoritative learner state: resuming the session reports it (README §4/§10).
func TestBeginStoresGradeBandOnResume(t *testing.T) {
	handler := api.NewHandler(session.NewMemoryStore())

	if rec := postJSON(t, handler, http.MethodPost, "/learners/Amaka/begin", map[string]string{
		"subject":   "Mathematics",
		"topic":     "Addition",
		"gradeBand": "JSS1-3",
	}); rec.Code != http.StatusOK {
		t.Fatalf("expected begin to succeed, got %d", rec.Code)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/learners/Amaka/session", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected resume to succeed, got %d", rec.Code)
	}

	var got struct {
		GradeBand string `json:"gradeBand"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("expected a JSON body, got error: %v", err)
	}
	if got.GradeBand != "JSS1-3" {
		t.Fatalf("expected grade band JSS1-3 on resume, got %q", got.GradeBand)
	}
}

// TestBeginRequiresGradeBand confirms begin now demands a grade band (it is
// needed for calibration), returning 400 when one is missing.
func TestBeginRequiresGradeBand(t *testing.T) {
	handler := api.NewHandler(session.NewMemoryStore())

	rec := postJSON(t, handler, http.MethodPost, "/learners/Amaka/begin", map[string]string{
		"subject": "Mathematics",
		"topic":   "Addition",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when gradeBand is missing, got %d", rec.Code)
	}
}

// TestBeginRejectsUnknownGradeBand confirms an undefined band is rejected
// (422) and creates no session (README §4 bands are safety requirements).
func TestBeginRejectsUnknownGradeBand(t *testing.T) {
	handler := api.NewHandler(session.NewMemoryStore())

	rec := postJSON(t, handler, http.MethodPost, "/learners/Amaka/begin", map[string]string{
		"subject":   "Mathematics",
		"topic":     "Addition",
		"gradeBand": "Grade 9",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for an unknown grade band, got %d", rec.Code)
	}

	resume := httptest.NewRecorder()
	handler.ServeHTTP(resume, httptest.NewRequest(http.MethodGet, "/learners/Amaka/session", nil))
	if resume.Code != http.StatusNotFound {
		t.Fatalf("a rejected begin must not create a session; expected 404 on resume, got %d", resume.Code)
	}
}
