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

const timeOutputLayout = "2006-01-02T15:04:05-07:00"

// ProductActivity holds signals specific to APIM. Distinct shape from
// Asgardeo's ProductActivity per team decision (2026-09-09) — different
// products, different concepts.
type ProductActivity struct {
	SignedIn          bool `json:"signedIn"`
	ProjectCreated    bool `json:"projectCreated"`
	ComponentCreated  bool `json:"componentCreated"`
	QuickStartSkipped bool `json:"quickStartSkipped"`
	// HasMeaningfulActivity uses a simple request-count threshold for now
	// (500+), per team decision (2026-09-16) — ASSUMPTION (not yet
	// confirmed): counts ALL events found, including any non-product
	// tracking/telemetry noise, since real per-product filtering wasn't
	// specified. To be refined based on which components were configured,
	// per the team's own stated plan.
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

// meaningfulActivityThreshold: see HasMeaningfulActivity's doc comment
// above for the assumption this rests on.
const meaningfulActivityThreshold = 500

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
