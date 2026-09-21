package apim

import "time"

// Action name values for APIM's "action_name" field — CONFIRMED real
// values shared by the team (2026-09-16). Only a subset are currently
// wired into Normalize()'s classification below; the rest are kept here
// as confirmed reference for when we build them in.
const (
	ActionNameLandingSignInSucceeded    = "Landing-SignIn-Succeeded"
	ActionNameQuickStartSkipped         = "QuickStart-Skipped"
	ActionNameProjectCreatedStart       = "Project-Created-Start"
	ActionNameComponentCreatedStart     = "Component-Created-Start"
	ActionNameQuickStartSelectedProduct = "QuickStart-Selected-Product"
)

// sriLankaLocation / timeOutputLayout: same Sri Lanka display convention
// as internal/moesif (team decision, 2026-09-16) — kept as a separate
// copy in this package rather than importing internal/moesif, since the
// two products are meant to stay independently buildable.
var sriLankaLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Colombo")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// timeOutputLayout: human-readable format per team decision (2026-09-16),
// e.g. "September 14, 2026 2:30 PM"
const timeOutputLayout = "January 2, 2006 3:04 PM"

// ProductActivity holds signals specific to APIM. Distinct shape from
// Asgardeo's ProductActivity per team decision (2026-09-09) — different
// products, different concepts.
type ProductActivity struct {
	SignedIn          bool `json:"signedIn"`
	ProjectCreated    bool `json:"projectCreated"`
	ComponentCreated  bool `json:"componentCreated"`
	QuickStartSkipped bool `json:"quickStartSkipped"`
	// HasMeaningfulActivity uses a simple request-count threshold (400+),
	// confirmed by the team (2026-09-18) — she describes this as "all api
	// requests being invoked" through the platform. STILL AN OPEN GAP: our
	// raw-data investigation found eventsFound also includes non-API-
	// request items (ad tracking pixels, telemetry pings) — worth
	// confirming whether she's aware of this, or whether it needs
	// filtering. To be refined further based on which components were
	// configured, per her own stated plan.
	HasMeaningfulActivity bool `json:"hasMeaningfulActivity"`
}

// Summary is APIM's normalized signal set. Parent-level fields (name,
// tenure, location) stay consistent with Asgardeo's Summary shape, per
// team decision (2026-09-09) that these fields are shared across products.
type Summary struct {
	OrganizationName string          `json:"organizationName,omitempty"`
	FirstSeen        string          `json:"firstSeen"`
	LastActivity     string          `json:"lastActivity"`
	Timezone         string          `json:"timezone,omitempty"`
	CountryName      string          `json:"countryName,omitempty"`
	IsWSO2User       bool            `json:"isWSO2User"`
	ProductActivity  ProductActivity `json:"productActivity"`
	EventsFound      int             `json:"eventsFound"`
}

// invoked through the platform. See HasMeaningfulActivity's doc comment
// for a caveat: our RAW data investigation found eventsFound also
// includes non-API-request items (ad tracking pixels, telemetry pings),
// which may not match what she has in mind — worth confirming.
const meaningfulActivityThreshold = 400

// Normalize aggregates raw APIM hits (already filtered to a single
// company/user) into a Summary.
func Normalize(hits []RawHit, total int) Summary {
	var summary Summary
	var earliest, latest time.Time

	for _, hit := range hits {
		src := hit.Source

		if summary.OrganizationName == "" && src.Company.Metadata.Name != "" {
			summary.OrganizationName = src.Company.Metadata.Name
		}
		if src.User.Metadata.IsWSO2User == "true" {
			summary.IsWSO2User = true
		}

		switch src.ActionName {
		case ActionNameLandingSignInSucceeded:
			summary.ProductActivity.SignedIn = true
		case ActionNameProjectCreatedStart:
			summary.ProductActivity.ProjectCreated = true
		case ActionNameComponentCreatedStart:
			summary.ProductActivity.ComponentCreated = true
		case ActionNameQuickStartSkipped:
			summary.ProductActivity.QuickStartSkipped = true
		}

		eventTime, timeErr := parseAPIMTime(src.Request.Time)
		if timeErr == nil {
			if eventTime.After(latest) {
				latest = eventTime
				summary.Timezone = src.Request.GeoIP.Timezone
				summary.CountryName = src.Request.GeoIP.CountryName
			}
			if earliest.IsZero() || eventTime.Before(earliest) {
				earliest = eventTime
			}
		}
	}

	if !latest.IsZero() {
		summary.LastActivity = latest.In(sriLankaLocation).Format(timeOutputLayout)
	}
	if !earliest.IsZero() {
		summary.FirstSeen = earliest.In(sriLankaLocation).Format(timeOutputLayout)
	}

	summary.EventsFound = total
	summary.ProductActivity.HasMeaningfulActivity = total >= meaningfulActivityThreshold

	return summary
}

// parseAPIMTime parses APIM's observed request.time format — CONFIRMED
// same "no timezone suffix" shape as Asgardeo's (2026-09-16).
func parseAPIMTime(raw string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05.000", raw)
}
