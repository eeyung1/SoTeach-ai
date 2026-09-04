package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"soteach/api"
	"soteach/session"
)

// TestDiagnosisReportEndpointReturnsFinding proves GET
// /learners/{name}/diagnosis surfaces what diagnosis found — the funnel's
// free value moment. After a normal (no-provider) diagnosis the report
// carries the learner's own words, subject/topic, and grade band.
func TestDiagnosisReportEndpointReturnsFinding(t *testing.T) {
	handler := api.NewHandler(session.NewMemoryStore())

	if rec := postJSON(t, handler, http.MethodPost, "/learners/Amaka/begin", map[string]string{
		"subject":   "Mathematics",
		"topic":     "Addition",
		"gradeBand": "JSS1-3",
	}); rec.Code != http.StatusOK {
		t.Fatalf("expected begin to succeed, got %d", rec.Code)
	}
	if rec := postJSON(t, handler, http.MethodPost, "/learners/Amaka/input", map[string]string{
		"input": "I can add single digits",
	}); rec.Code != http.StatusOK {
		t.Fatalf("expected the diagnostic answer to be accepted, got %d", rec.Code)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/learners/Amaka/diagnosis", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a completed diagnosis, got %d", rec.Code)
	}

	var got struct {
		Finding   string `json:"finding"`
		Topic     string `json:"topic"`
		GradeBand string `json:"gradeBand"`
		Concept   string `json:"concept"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("expected a JSON body, got error: %v", err)
	}
	if got.Finding != "I can add single digits" {
		t.Fatalf("expected the verbatim finding, got %q", got.Finding)
	}
	if got.Topic != "Addition" {
		t.Fatalf("expected topic Addition, got %q", got.Topic)
	}
	if got.GradeBand != "JSS1-3" {
		t.Fatalf("expected grade band JSS1-3, got %q", got.GradeBand)
	}
	if got.Concept != "" {
		t.Fatalf("expected no concept without an AI evaluation, got %q", got.Concept)
	}
}

// TestDiagnosisReportEndpointRejectsUnknownLearner maps a learner with no
// session to 404.
func TestDiagnosisReportEndpointRejectsUnknownLearner(t *testing.T) {
	handler := api.NewHandler(session.NewMemoryStore())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/learners/Bello/diagnosis", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for an unknown learner, got %d", rec.Code)
	}
}

// TestDiagnosisReportEndpointRejectedBeforeDiagnosisComplete maps a report
// request before diagnosis is finished to a conflict (the learner has a
// session, but nothing has been diagnosed yet).
func TestDiagnosisReportEndpointRejectedBeforeDiagnosisComplete(t *testing.T) {
	handler := api.NewHandler(session.NewMemoryStore())

	if rec := postJSON(t, handler, http.MethodPost, "/learners/Amaka/begin", map[string]string{
		"subject":   "Mathematics",
		"topic":     "Addition",
		"gradeBand": "JSS1-3",
	}); rec.Code != http.StatusOK {
		t.Fatalf("expected begin to succeed, got %d", rec.Code)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/learners/Amaka/diagnosis", nil))
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 before diagnosis is complete, got %d", rec.Code)
	}
}
