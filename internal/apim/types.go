package apim

// SearchResponse mirrors Moesif's Search Events API response envelope —
// CONFIRMED identical structure to the moesif package's Asgardeo version
// (2026-09-16): same hits.hits/hits.total nesting.
type SearchResponse struct {
	Result   HitsResult `json:"hits"`
	Took     int        `json:"took"`
	TimedOut bool       `json:"timed_out"`
}

type HitsResult struct {
	Hits  []RawHit `json:"hits"`
	Total int      `json:"total"`
}

type RawHit struct {
	ID     string        `json:"_id"`
	Source RawSource     `json:"_source"`
	Sort   []interface{} `json:"sort"`
}

// RawSource is APIM's actual event data — CONFIRMED against real
// api-platform (console.bijira.dev) events (2026-09-16). Differs from
// Asgardeo's RawSource in real, confirmed ways: company name lives at
// company.metadata.name (not account_name), and there's a direct
// isWSO2User flag (Asgardeo has no equivalent — we infer via domain
// instead).
type RawSource struct {
	CompanyID  string      `json:"company_id"`
	UserID     string      `json:"user_id"`
	EventType  string      `json:"event_type"` // observed: "user_action" on named events
	ActionName string      `json:"action_name"`
	Request    RawRequest  `json:"request"`
	Company    RawCompany  `json:"company"`
	User       RawUser     `json:"user"`
	Metadata   RawMetadata `json:"metadata"`
}

type RawRequest struct {
	Time  string   `json:"time"`
	GeoIP RawGeoIP `json:"geo_ip"`
}

type RawGeoIP struct {
	Timezone    string `json:"timezone"`
	CountryName string `json:"country_name"`
}

type RawCompany struct {
	Metadata RawCompanyMetadata `json:"metadata"`
}

// RawCompanyMetadata: CONFIRMED real field is "name", NOT "account_name"
// like Asgardeo (2026-09-16).
type RawCompanyMetadata struct {
	Name string `json:"name"`
}

// RawUser: isWSO2User REMOVED per team decision (2026-09-24) — APIM now
// uses the same domain-based WSO2 check as every other product, not a
// product-specific direct flag, for consistency.
type RawUser struct {
	Email string `json:"email"`
}

// RawMetadata holds event-specific metadata, confirmed by the team
// (2026-09-22) to carry these fields on specific quick-start events:
//   - QuickStart-Attempted-Source: Method
//   - QuickStart-Validation: Source, Outcome
//   - QuickStart-Selected-Source: Source
//
// DeploymentModel (from QuickStart-Selected-Product, the "SAS" vs
// "gateway" choice) is NOT YET CONFIRMED — the real field name it lives
// under wasn't specified in the meeting. This is a PLACEHOLDER guess,
// not verified against real data. Confirm the actual field name before
// relying on this.
//
// RawMetadata: DeploymentModel CONFIRMED real (2026-09-23), real value
// seen: "saas". Context is NEW (2026-09-24) — a separate metadata value
// on QuickStart-Selected-Product, alongside deployment_model. Field name
// assumed to literally be "context"; not yet verified against real data.
type RawMetadata struct {
	Method          string `json:"method"`
	Source          string `json:"source"`
	Outcome         string `json:"outcome"`
	DeploymentModel string `json:"deployment_model"`
	Context         string `json:"context"`
}
