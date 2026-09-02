package ai

// FakeAIProvider is a test double for AIProvider (Agent.md §43): it
// returns a predetermined DiagnosticResult instead of calling a live
// model, so domain tests do not depend on network access, API
// credentials, provider latency, or model randomness.
type FakeAIProvider struct {
	DiagnoseResult DiagnosticResult
	DiagnoseErr    error
}

// Diagnose returns the configured DiagnoseResult/DiagnoseErr, ignoring
// its arguments. Tests configure the fields directly before calling it.
func (f *FakeAIProvider) Diagnose(concept, learnerResponse string) (DiagnosticResult, error) {
	return f.DiagnoseResult, f.DiagnoseErr
}
