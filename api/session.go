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
	"soteach/tutor"
)

// NewHandler wires the API's HTTP routes to the given session store and
// returns a handler ready to serve. REST/JSON over the Go standard library,
// the single API style for this project (Agent.md §19).
func NewHandler(store *session.MemoryStore) http.Handler {
	a := &sessionAPI{store: store, tutor: tutor.NewTutor(store)}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /learners/{learner}/session", a.resumeSession)
	mux.HandleFunc("POST /learners/{learner}/begin", a.beginSession)
	mux.HandleFunc("POST /learners/{learner}/input", a.submitInput)
	return mux
}

type sessionAPI struct {
	store *session.MemoryStore
	tutor *tutor.Tutor
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

// beginRequest is the body of POST /learners/{learner}/begin.
type beginRequest struct {
	Subject string `json:"subject"`
	Topic   string `json:"topic"`
}

// inputRequest is the body of POST /learners/{learner}/input.
type inputRequest struct {
	Input string `json:"input"`
}

// actionResponse is what every tutoring action returns: the prompt to show
// the learner plus the authoritative state token that follows, so a client
// knows whether the prompt is a question, an explanation, or a completion.
type actionResponse struct {
	Prompt string `json:"prompt"`
	State  string `json:"state"`
}

// beginSession implements POST /learners/{learner}/begin: it starts (or
// resumes) a learner's session on a topic and returns the first thing the
// learner should respond to. All decisions belong to the tutor; this handler
// only decodes the request and encodes the result (Agent.md §19).
func (a *sessionAPI) beginSession(w http.ResponseWriter, r *http.Request) {
	learner := r.PathValue("learner")

	var req beginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}
	if req.Subject == "" || req.Topic == "" {
		writeError(w, http.StatusBadRequest, "subject and topic are required")
		return
	}

	outcome, err := a.tutor.BeginTopic(learner, req.Subject, req.Topic)
	a.respondAction(w, learner, outcome, err)
}

// submitInput implements POST /learners/{learner}/input: it applies the
// learner's latest input (explanation or answer) to their session and returns
// the next prompt. Routing is the tutor's decision, not this handler's.
func (a *sessionAPI) submitInput(w http.ResponseWriter, r *http.Request) {
	learner := r.PathValue("learner")

	var req inputRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}
	if req.Input == "" {
		writeError(w, http.StatusBadRequest, "input is required")
		return
	}

	outcome, err := a.tutor.ApplyInput(learner, req.Input)
	a.respondAction(w, learner, outcome, err)
}

// respondAction encodes a tutor action result: on error it maps the domain
// error to a status code; on success it returns the prompt plus the resulting
// state token (read back from the store, which the action just updated).
func (a *sessionAPI) respondAction(w http.ResponseWriter, learner string, outcome tutor.Outcome, err error) {
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}

	s, loadErr := a.store.Load(learner)
	if loadErr != nil {
		writeError(w, http.StatusInternalServerError, loadErr.Error())
		return
	}

	writeJSON(w, http.StatusOK, actionResponse{Prompt: outcome.Prompt, State: stateName(s.State())})
}

// statusForError maps a domain/tutor error to the HTTP status a client should
// see: 404 for an unknown learner, 409 when the session is in a state that
// cannot accept the request, 422 for unsupported content, 500 otherwise.
func statusForError(err error) int {
	switch {
	case errors.Is(err, session.ErrSessionNotFound):
		return http.StatusNotFound
	case errors.Is(err, tutor.ErrNoQuestionPending),
		errors.Is(err, tutor.ErrNothingAwaitingInput),
		errors.Is(err, tutor.ErrSessionNotAwaitingDiagnosis):
		return http.StatusConflict
	case errors.Is(err, tutor.ErrTopicNotYetSupported):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}
