package email

// Generator produces a personalized outreach email from prospect context.
// A real implementation calls the Anthropic API; a mock implementation is
// used in tests until real API access is available (pending as of
// 2026-09-15).
type Generator interface {
	Generate(prompt Prompt) (string, error)
}

// Prompt holds everything needed to generate one email — kept as a
// struct (not raw strings) so it's easy to see everything Claude will be
// given, and callers can't accidentally mix up argument order.
type Prompt struct {
	OrganizationName string
	Outcome          string // e.g. "PLG CS Eligible"
	Tags             []string
	ActivitySummary  string // plain-text description of the prospect's known activity
	EmailTemplate    string
}
