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
// subjects with their topics and the defined grade bands — Mathematics with its
// addition/subtraction/multiplication/division topics plus English Language's
// first topic, over the three grade bands (Primary 4-6, JSS1-3, SSS1-3).
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

	if len(got.Subjects) != 2 {
		t.Fatalf("expected two subjects, got %d", len(got.Subjects))
	}
	if got.Subjects[0].Name != "Mathematics" {
		t.Fatalf("expected subject Mathematics first, got %q", got.Subjects[0].Name)
	}
	wantMath := []string{"Addition", "Subtraction", "Multiplication", "Division"}
	if !equalStrings(got.Subjects[0].Topics, wantMath) {
		t.Fatalf("expected Mathematics topics %v, got %v", wantMath, got.Subjects[0].Topics)
	}
	if got.Subjects[1].Name != "English Language" {
		t.Fatalf("expected subject English Language second, got %q", got.Subjects[1].Name)
	}
	if !equalStrings(got.Subjects[1].Topics, []string{"Parts of Speech"}) {
		t.Fatalf("expected English Language topics [Parts of Speech], got %v", got.Subjects[1].Topics)
	}

	if len(got.GradeBands) != 3 {
		t.Fatalf("expected three grade bands, got %v", got.GradeBands)
	}
}

// equalStrings reports whether a and b hold the same strings in the same order.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
