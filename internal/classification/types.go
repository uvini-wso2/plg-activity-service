package classification

// Response mirrors the real classification API's response shape —
// CONFIRMED against the real staging endpoint (2026-09-29/30), verified
// via both Postman and curl. Note: "signals" fields are camelCase in the
// real API (syntaxValid, mxValid, freeProvider, roleBased) — this differs
// from an earlier, incorrect assumption (snake_case) used elsewhere in
// this codebase before this endpoint was confirmed.
type Response struct {
	Email      string  `json:"email"`
	Domain     string  `json:"domain"`
	Category   string  `json:"category"`
	Rating     int     `json:"rating"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"`
	Signals    Signals `json:"signals"`
	Reasoning  string  `json:"reasoning"`
}

type Signals struct {
	SyntaxValid  bool `json:"syntaxValid"`
	MXValid      bool `json:"mxValid"`
	Disposable   bool `json:"disposable"`
	FreeProvider bool `json:"freeProvider"`
	RoleBased    bool `json:"roleBased"`
}

// Category constants. Only "corporate" has been directly confirmed
// against the real live endpoint so far (2026-09-30) — the others match
// the examples shared by the team weeks ago, but haven't been verified
// against this specific real endpoint yet.
const (
	CategoryCorporate       = "corporate"
	CategoryPersonal        = "personal"
	CategoryProviderTesting = "provider_testing"
	CategoryDisposable      = "disposable"
)
