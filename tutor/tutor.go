// Package tutor is the thin wiring layer between an ai.AIProvider and a
// session.Session (Agent.md §42). It exists so the AI request itself
// happens outside the tutoring engine — session.Session never calls an
// AIProvider directly — while the validation and state transition still
// happen inside session.Session, via its existing RecordDiagnosticResult
// boundary (Agent.md §40).
package tutor

import (
	"soteach/ai"
	"soteach/session"
)

// DiagnoseWithAI calls provider.Diagnose with the learner's raw
// diagnostic response, then hands the result to
// Session.RecordDiagnosticResult, which validates it before it can
// change any state.
//
// A provider failure (Agent.md §41 — timeout, network failure, malformed
// output, etc.) is returned as-is and never reaches Session: the session
// is left exactly as it was, recoverable, per §42 ("a provider failure
// must not corrupt learner state"). A validation failure on an
// otherwise-successful provider call is likewise returned, and likewise
// leaves Session untouched, since RecordDiagnosticResult itself never
// stores an invalid result.
func DiagnoseWithAI(s *session.Session, provider ai.AIProvider, concept, learnerResponse string) error {
	result, err := provider.Diagnose(concept, learnerResponse)
	if err != nil {
		return err
	}

	return s.RecordDiagnosticResult(result)
}
