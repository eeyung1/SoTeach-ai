package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"soteach/api"
	"soteach/session"
)

// TestResumeSessionEndpointReturnsSavedState proves the API boundary
// (Agent.md §19; workingReadme.md §3.2/§9) for the resume rule in README §6:
// a session interrupted mid-loop and saved to the store can be resumed
// through HTTP, returning the learner's current state — subject, topic,
// state-machine step, and the pending question — as JSON, without repeating
// the diagnostic stage. The handler only translates; the state it returns
// comes from the domain Session via the store.
func TestResumeSessionEndpointReturnsSavedState(t *testing.T) {
	store := session.NewMemoryStore()

	original := session.New("Amaka")
	if err := original.SetGradeBand("JSS1-3"); err != nil {
		t.Fatalf("expected grade band to be set, got error: %v", err)
	}
	original.SelectSubject("Mathematics")
	if err := original.SelectTopic("Addition"); err != nil {
		t.Fatalf("expected topic to be selected, got error: %v", err)
	}
	original.StartDiagnosis("What do you already know about addition?")
	original.RecordDiagnosticResponse("I know how to add single digits")
	if err := original.AskQuestion("What is 7 + 3?", "10"); err != nil {
		t.Fatalf("expected question to be asked, got error: %v", err)
	}
	store.Save(original)

	handler := api.NewHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/learners/Amaka/session", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for a saved session, got %d", rec.Code)
	}

	var got struct {
		Learner           string `json:"learner"`
		GradeBand         string `json:"gradeBand"`
		Subject           string `json:"subject"`
		Topic             string `json:"topic"`
		State             string `json:"state"`
		CurrentQuestion   string `json:"currentQuestion"`
		DiagnosisComplete bool   `json:"diagnosisComplete"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("expected a JSON body, got error: %v", err)
	}

	if got.Learner != "Amaka" {
		t.Fatalf("expected learner Amaka, got %q", got.Learner)
	}
	if got.GradeBand != "JSS1-3" {
		t.Fatalf("expected grade band JSS1-3, got %q", got.GradeBand)
	}
	if got.Subject != "Mathematics" {
		t.Fatalf("expected subject Mathematics, got %q", got.Subject)
	}
	if got.Topic != "Addition" {
		t.Fatalf("expected topic Addition, got %q", got.Topic)
	}
	if got.State != "wait_for_answer" {
		t.Fatalf("expected state wait_for_answer, got %q", got.State)
	}
	if got.CurrentQuestion != "What is 7 + 3?" {
		t.Fatalf("expected the pending question to be returned, got %q", got.CurrentQuestion)
	}
	if !got.DiagnosisComplete {
		t.Fatal("expected diagnosisComplete true — resuming must not re-open completed diagnosis")
	}
}

// TestResumeSessionEndpointUnknownLearnerReturns404 confirms the API maps an
// absent saved session to a 404, so a caller can distinguish "resume this"
// from "no session exists for this learner".
func TestResumeSessionEndpointUnknownLearnerReturns404(t *testing.T) {
	store := session.NewMemoryStore()
	handler := api.NewHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/learners/Bello/session", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a learner with no saved session, got %d", rec.Code)
	}
}
