package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"soteach/api"
	"soteach/session"
)

// postJSON is a helper: sends an HTTP request with a JSON body to the API
// handler and returns the recorder.
func postJSON(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("expected request body to marshal, got error: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// decodeAction decodes an action response ({prompt, state}) from a recorder.
func decodeAction(t *testing.T, rec *httptest.ResponseRecorder) (prompt, state string) {
	t.Helper()
	var got struct {
		Prompt string `json:"prompt"`
		State  string `json:"state"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("expected a JSON body, got error: %v", err)
	}
	return got.Prompt, got.State
}

// TestBeginEndpointReturnsDiagnosticPrompt proves the full entry path over
// HTTP: POST begin creates the session server-side and returns the diagnostic
// prompt with the diagnosing state, so a client can immediately render what
// to ask the learner.
func TestBeginEndpointReturnsDiagnosticPrompt(t *testing.T) {
	store := session.NewMemoryStore()
	handler := api.NewHandler(store)

	rec := postJSON(t, handler, http.MethodPost, "/learners/Amaka/begin", map[string]string{
		"subject":   "Mathematics",
		"topic":     "Addition",
		"gradeBand": "JSS1-3",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	prompt, state := decodeAction(t, rec)
	if prompt != "What do you already know about addition?" {
		t.Fatalf("expected the diagnostic prompt, got %q", prompt)
	}
	if state != "diagnosing" {
		t.Fatalf("expected state diagnosing, got %q", state)
	}
}

// TestInputEndpointDrivesLoopToMastery is the money test for the API boundary
// (workingReadme §8, Stage 1 done-condition): the entire tutoring loop —
// diagnose, practice, verify, mastery — driven end to end over HTTP, with the
// client sending only the learner's words and reading back a prompt plus the
// authoritative state token.
func TestInputEndpointDrivesLoopToMastery(t *testing.T) {
	store := session.NewMemoryStore()
	handler := api.NewHandler(store)

	if rec := postJSON(t, handler, http.MethodPost, "/learners/Amaka/begin", map[string]string{
		"subject":   "Mathematics",
		"topic":     "Addition",
		"gradeBand": "JSS1-3",
	}); rec.Code != http.StatusOK {
		t.Fatalf("expected begin to succeed, got %d", rec.Code)
	}

	diag, state := decodeAction(t, postJSON(t, handler, http.MethodPost, "/learners/Amaka/input", map[string]string{
		"input": "I can add single digits",
	}))
	if diag != "What is 7 + 3?" {
		t.Fatalf("expected the first practice question, got %q", diag)
	}
	if state != "wait_for_answer" {
		t.Fatalf("expected state wait_for_answer after diagnosis, got %q", state)
	}

	transfer, state := decodeAction(t, postJSON(t, handler, http.MethodPost, "/learners/Amaka/input", map[string]string{
		"input": "10",
	}))
	if transfer != "What is 5 + 6?" {
		t.Fatalf("expected a transfer question after the correct answer, got %q", transfer)
	}
	if state != "wait_for_answer" {
		t.Fatalf("expected state wait_for_answer with the transfer question pending, got %q", state)
	}

	completion, state := decodeAction(t, postJSON(t, handler, http.MethodPost, "/learners/Amaka/input", map[string]string{
		"input": "11",
	}))
	if completion != "That's right! You have mastered Addition." {
		t.Fatalf("expected the completion message, got %q", completion)
	}
	if state != "mastered" {
		t.Fatalf("expected state mastered, got %q", state)
	}
}

// TestBeginEndpointRejectsUnsupportedTopic confirms unsupported curriculum
// content maps to 422 and does not create a session (README §5).
func TestBeginEndpointRejectsUnsupportedTopic(t *testing.T) {
	store := session.NewMemoryStore()
	handler := api.NewHandler(store)

	rec := postJSON(t, handler, http.MethodPost, "/learners/Amaka/begin", map[string]string{
		"subject":   "Mathematics",
		"topic":     "Fractions",
		"gradeBand": "JSS1-3",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for an unsupported topic, got %d", rec.Code)
	}

	resume := httptest.NewRecorder()
	handler.ServeHTTP(resume, httptest.NewRequest(http.MethodGet, "/learners/Amaka/session", nil))
	if resume.Code != http.StatusNotFound {
		t.Fatalf("a rejected begin must not create a session; expected 404 on resume, got %d", resume.Code)
	}
}

// TestInputEndpointRejectsUnknownLearner confirms input for a learner with no
// begun session maps to 404 (they must begin first), not a silent no-op.
func TestInputEndpointRejectsUnknownLearner(t *testing.T) {
	store := session.NewMemoryStore()
	handler := api.NewHandler(store)

	rec := postJSON(t, handler, http.MethodPost, "/learners/Bello/input", map[string]string{
		"input": "hello",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a learner with no begun session, got %d", rec.Code)
	}
}

// TestBeginEndpointRejectsMalformedBody confirms a body that is not valid JSON
// maps to 400 rather than being silently ignored.
func TestBeginEndpointRejectsMalformedBody(t *testing.T) {
	store := session.NewMemoryStore()
	handler := api.NewHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/learners/Amaka/begin", bytes.NewReader([]byte("{not json")))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a malformed body, got %d", rec.Code)
	}
}
