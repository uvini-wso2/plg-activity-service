package validation

// EmailClassification mirrors the classification API's response shape.
// CONFIRMED (2026-09-11) — real example responses shared by the team.
type EmailClassification struct {
	Email      string  `json:"email"`
	Domain     string  `json:"domain"`
	Category   string  `json:"category"` // "corporate", "personal", "provider_testing", "disposable"
	Rating     int     `json:"rating"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"` // "llm" or "rules"
	Signals    Signals `json:"signals"`
	Reasoning  string  `json:"reasoning"`
}

type Signals struct {
	SyntaxValid  bool `json:"syntax_valid"`
	MXValid      bool `json:"mx_valid"`
	Disposable   bool `json:"disposable"`
	FreeProvider bool `json:"free_provider"`
	RoleBased    bool `json:"role_based"`
}

// Category constants — confirmed real values from the classification API.
const (
	CategoryCorporate       = "corporate"
	CategoryPersonal        = "personal"
	CategoryProviderTesting = "provider_testing"
	CategoryDisposable      = "disposable"
)

// Outcome is the three-way PLG CS classification result, per team decision
// (2026-09-11).
type Outcome string

const (
	OutcomeEligible  Outcome = "PLG CS Eligible"
	OutcomeExcluded  Outcome = "PLG CS Excluded"
	OutcomeMonitored Outcome = "PLG CS Monitored"
)

// Tag names — confirmed exact set given by the team (2026-09-11). Do not
// invent new tags; extend this list only on explicit confirmation.
const (
	TagCorporateDomain           = "Corporate Domain"
	TagMeaningfulProductActivity = "Meaningful Product Activities"
	TagExplicitNeedForAssistance = "Explicit Need for Assistance" // NOTE: data source for this signal is out of scope for now (2026-09-11) — no logic currently sets this tag.
	TagDisposableDomain          = "Disposable Domain"
	TagWSO2Domain                = "WSO2 Domain"
	TagInvalidEmail              = "Invalid Email"
)

// Result is the full validation output: the outcome plus the tags that
// justify it, so a CS engineer can see WHY a decision was made.
type Result struct {
	Outcome Outcome  `json:"outcome"`
	Tags    []string `json:"tags"`
}
