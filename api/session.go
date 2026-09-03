// Package api is the thin HTTP translation layer for the tutoring engine
// (Agent.md §19; workingReadme.md §3.2). No tutoring logic lives here:
// handlers translate HTTP requests into domain operations on package session
// (via the store) and domain results back into HTTP responses. The domain
// (session.Session / session.MemoryStore) stays fully testable without HTTP.
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"soteach/session"
)

// NewHandler wires the API's HTTP routes to the given session store and
// returns a handler ready to serve. REST/JSON over the Go standard library,
// the single API style for this project (Agent.md §19).
func NewHandler(store *session.MemoryStore) http.Handler {
	a := &sessionAPI{store: store}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /learners/{learner}/session", a.resumeSession)
	return mux
}

type sessionAPI struct {
	store *session.MemoryStore
}

// sessionState is the JSON representation of a resumed session: everything a
// client needs to continue rendering the state machine (workingReadme §3.2).
type sessionState struct {
	Learner           string `json:"learner"`
	GradeBand         string `json:"gradeBand"`
	Subject           string `json:"subject"`
	Topic             string `json:"topic"`
	State             string `json:"state"`
	CurrentQuestion   string `json:"currentQuestion"`
	DiagnosisComplete bool   `json:"diagnosisComplete"`
}

// resumeSession implements GET /learners/{learner}/session (README §6
// resuming an unfinished session). It loads the learner's saved session from
// the store and returns its current state; an absent session maps to 404.
// The handler performs no state transitions — it only reads and translates.
func (a *sessionAPI) resumeSession(w http.ResponseWriter, r *http.Request) {
	learner := r.PathValue("learner")

	s, err := a.store.Load(learner)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, session.ErrSessionNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toSessionState(s))
}

// toSessionState maps a domain Session to its JSON representation.
func toSessionState(s *session.Session) sessionState {
	return sessionState{
		Learner:           s.LearnerName(),
		GradeBand:         s.GradeBand(),
		Subject:           s.Subject(),
		Topic:             s.Topic(),
		State:             stateName(s.State()),
		CurrentQuestion:   s.CurrentQuestion(),
		DiagnosisComplete: s.DiagnosisComplete(),
	}
}

// stateName is the stable string token for a state-machine step, so clients
// are not coupled to the integer State values.
func stateName(st session.State) string {
	switch st {
	case session.Idle:
		return "idle"
	case session.Diagnosing:
		return "diagnosing"
	case session.WaitForAnswer:
		return "wait_for_answer"
	case session.CheckAnswer:
		return "check_answer"
	case session.AwaitingTransferCheck:
		return "awaiting_transfer_check"
	case session.Incorrect:
		return "incorrect"
	case session.Uncertain:
		return "uncertain"
	case session.Mastered:
		return "mastered"
	default:
		return "unknown"
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
