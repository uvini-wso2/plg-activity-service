package moesif

import "time"

// Action name values for Moesif's "action_name" field.
const (
	// CONFIRMED — observed in real Moesif responses for Asgardeo activity,
	// across Prod/Dev/Staging/Test environments (2026-09-08/15).
	ActionNameOrganizationCreated     = "organization_created"
	ActionNameOrganizationSubscribed  = "organization_subscribed"
	ActionNameUserCreated             = "user_created"
	ActionNameOnboardingStepCompleted = "Onboarding-Step-Completed"
	ActionNameOnboardingSkipped       = "Onboarding-Skipped"
	// ActionNameOnboardingCompleted marks genuine FULL completion of the
	// onboarding wizard — distinct from ApplicationCreated (which only
	// means at least one step was done). CONFIRMED real value, seen in
	// the Prod dashboard's action list (2026-09-08), not wired up until
	// now (2026-09-15).
	ActionNameOnboardingCompleted = "Onboarding-Completed"

	// CONFIRMED UNAVAILABLE for Asgardeo: authentication/login events are
	// NOT tracked for Asgardeo's own product analytics. Kept as reference
	// for another product's classification logic.
	ActionNameAuthenticationAttempt = "authentication_attempt"
	ActionNameAPICall               = "api_call"
)

// sriLankaLocation is used to format ALL output timestamps in Sri Lankan
// time, since the CS team operating this system is based in Sri Lanka
// (team decision, 2026-09-16). This is separate from the "timezone" and
// "countryName" fields below, which report the PROSPECT's own location —
// this only affects how firstSeen/lastActivity are DISPLAYED. Falls back
// to UTC if the timezone database is somehow unavailable.
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

// ProductActivity holds signals specific to THIS product (Asgardeo).
type ProductActivity struct {
	ApplicationCreated bool `json:"applicationCreated"`
	// HasCompletedOnboarding is true only when a genuine
	// Onboarding-Completed event occurred — the full wizard finished, not
	// just one step. Distinct from ApplicationCreated (any single step
	// done) and independent of SkippedStepNumber below (2026-09-15).
	HasCompletedOnboarding bool `json:"hasCompletedOnboarding"`
	// SkippedStepNumber / SkippedStepName still track the most recent
	// Onboarding-Skipped event, if any — used by validation logic to
	// detect "started but stopped at an identifiable point". nil means no
	// skip occurred; a real, valid step (e.g. 0) is a genuine skip.
	SkippedStepNumber   *int   `json:"skippedStepNumber"`
	SkippedStepName     string `json:"skippedStepName,omitempty"`
	OnboardingSetupType string `json:"onboardingSetupType,omitempty"`
}

// Summary is the normalized, per-customer signal set. Fields here stay
// consistent across all products; product-specific signals live in
// ProductActivity instead.
type Summary struct {
	OrganizationName string          `json:"organizationName,omitempty"`
	FirstSeen        string          `json:"firstSeen"`
	LastActivity     string          `json:"lastActivity"`
	Timezone         string          `json:"timezone,omitempty"`
	CountryName      string          `json:"countryName,omitempty"`
	ProductActivity  ProductActivity `json:"productActivity"`
}

// Normalize aggregates a slice of raw Moesif hits (already filtered to a
// single company/user) into a Summary.
func Normalize(hits []RawHit) Summary {
	var summary Summary
	var earliest, latest time.Time
	var latestSkipTime time.Time

	for _, hit := range hits {
		src := hit.Source

		if summary.OrganizationName == "" && src.Company.Metadata.AccountName != "" {
			summary.OrganizationName = src.Company.Metadata.AccountName
		}
		if summary.ProductActivity.OnboardingSetupType == "" && src.Metadata.WizardPath != "" {
			summary.ProductActivity.OnboardingSetupType = src.Metadata.WizardPath
		}

		eventTime, timeErr := parseMoesifTime(src.Request.Time)

		switch src.ActionName {
		case ActionNameOnboardingStepCompleted:
			summary.ProductActivity.ApplicationCreated = true
		case ActionNameOnboardingCompleted:
			summary.ProductActivity.HasCompletedOnboarding = true
		case ActionNameOnboardingSkipped:
			if timeErr == nil && (latestSkipTime.IsZero() || eventTime.After(latestSkipTime)) {
				latestSkipTime = eventTime
				summary.ProductActivity.SkippedStepNumber = src.Metadata.StepNumber
				summary.ProductActivity.SkippedStepName = src.Metadata.StepName
			}
		}

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

	return summary
}

// parseMoesifTime parses the observed Moesif request.time format, e.g.
// "2026-09-07T02:00:58.646" — no timezone suffix, treated as UTC.
func parseMoesifTime(raw string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05.000", raw)
}
