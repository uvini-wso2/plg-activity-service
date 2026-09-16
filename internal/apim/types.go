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
	CompanyID  string     `json:"company_id"`
	UserID     string     `json:"user_id"`
	EventType  string     `json:"event_type"` // observed: "user_action" on named events
	ActionName string     `json:"action_name"`
	Request    RawRequest `json:"request"`
	Company    RawCompany `json:"company"`
	User       RawUser    `json:"user"`
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

type RawUser struct {
	Email    string          `json:"email"`
	Metadata RawUserMetadata `json:"metadata"`
}

// RawUserMetadata: CONFIRMED real field isWSO2User (string "true"/"false",
// not a real bool, per observed raw data) — a direct signal APIM has that
// Asgardeo does NOT; for Asgardeo we infer internal-vs-external purely
// from checking the email domain instead.
type RawUserMetadata struct {
	IsWSO2User string `json:"isWSO2User"`
}
