package apim

import "time"

// Action name values for APIM's "action_name" field — CONFIRMED real
// values, expanded per team decision (2026-09-22). Fields marked "not
// wired in" below are kept as reference only.
const (
	ActionNameLandingSignInSucceeded = "Landing-SignIn-Succeeded"
	ActionNameQuickStartSkipped      = "QuickStart-Skipped"

	// Confirmed real, but explicitly NOT tracked in ProductActivity per
	// team decision (2026-09-22) — kept as reference only:
	//   ActionNameLandingSignUpSucceeded = "Landing-SignUp-Succeeded"
	//   ActionNamePortalViewedHome        = "Portal-Viewed-Home"
	//   ActionNameHomePageVisit           = "home-page-visit"
	//   ActionNameProjectCreatedStart     = "Project-Created-Start"
	//   ActionNameLandingSignInViewed     = "Landing-SignIn-Viewed"
	//   ActionNameLandingSignInFailed     = "Landing-SignIn-Failed"
	//   ActionNameLandingViewedPage       = "Landing-Viewed-Page"
	//   ActionNameAPIInvoked              = "API-Invoked" // still under investigation by the team

	ActionNameQuickStartSelectedProduct = "QuickStart-Selected-Product"
	ActionNameComponentCreatedStart     = "Component-Created-Start"
	ActionNameComponentCreatedEnd       = "Component-Created-End"
	ActionNameQuickStartAttemptedSource = "QuickStart-Attempted-Source"
	ActionNameQuickStartValidation      = "QuickStart-Validation"
	ActionNameQuickStartSelectedSource  = "QuickStart-Selected-Source"
	ActionNameGatewayActivated          = "Gateway-Activated"
	ActionNameComponentDeployed         = "Component-Deployed"
	ActionNameComponentTested           = "Component-Tested"
	ActionNameComponentPromoted         = "Component-Promoted"
	ActionNameComponentGeneratedKey     = "Component-Generated-Key"
)

// sriLankaLocation / timeOutputLayout: same Sri Lanka display convention
// as internal/moesif (team decision, 2026-09-16).
var sriLankaLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Colombo")
	if err != nil {
		return time.UTC
	}
	return loc
}()

const timeOutputLayout = "January 2, 2006 3:04 PM"

// ProductActivity holds signals specific to APIM. Redefined per team
// decision (2026-09-22) — see normalize_test.go for confirmed behavior of
// each field.
type ProductActivity struct {
	// QuickStartCompleted: renamed + INVERTED from the old
	// "QuickStartSkipped" (2026-09-22) — true when NO skip event exists
	// (they went through fully), false when a skip event IS present.
	QuickStartCompleted bool `json:"quickStartCompleted"`

	QuickStartSelectedProduct bool `json:"quickStartSelectedProduct"`
	// DeploymentModel: from QuickStart-Selected-Product's metadata — see
	// RawMetadata.DeploymentModel's doc comment; field name UNCONFIRMED.
	DeploymentModel string `json:"deploymentModel,omitempty"`

	// APICreated is true ONLY when BOTH Component-Created-Start AND
	// Component-Created-End are present — confirmed by the team
	// (2026-09-22): "if both these events are there we take it as API is
	// created."
	APICreated bool `json:"apiCreated"`

	AttemptedSourceMethod string `json:"attemptedSourceMethod,omitempty"`
	ValidationSource      string `json:"validationSource,omitempty"`
	ValidationOutcome     string `json:"validationOutcome,omitempty"`
	SelectedSource        string `json:"selectedSource,omitempty"`

	GatewayActivated      bool `json:"gatewayActivated"`
	ComponentDeployed     bool `json:"componentDeployed"`
	ComponentTested       bool `json:"componentTested"`
	ComponentPromoted     bool `json:"componentPromoted"`
	ComponentKeyGenerated bool `json:"componentKeyGenerated"`

	// HasMeaningfulActivity: threshold confirmed 400 (2026-09-18), but
	// WHAT is counted is still unresolved as of 2026-09-22 — currently
	// counts every event Moesif returns, including confirmed tracking
	// noise and duplicate events. See README for details.
	HasMeaningfulActivity bool `json:"hasMeaningfulActivity"`
}

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

const meaningfulActivityThreshold = 400

// Normalize aggregates raw APIM hits (already filtered to a single
// company/user) into a Summary.
func Normalize(hits []RawHit, total int) Summary {
	var summary Summary
	var earliest, latest time.Time
	var quickStartSkipped, componentCreatedStart, componentCreatedEnd bool

	for _, hit := range hits {
		src := hit.Source

		if summary.OrganizationName == "" && src.Company.Metadata.Name != "" {
			summary.OrganizationName = src.Company.Metadata.Name
		}
		if src.User.Metadata.IsWSO2User == "true" {
			summary.IsWSO2User = true
		}

		switch src.ActionName {
		case ActionNameQuickStartSkipped:
			quickStartSkipped = true
		case ActionNameQuickStartSelectedProduct:
			summary.ProductActivity.QuickStartSelectedProduct = true
			if src.Metadata.DeploymentModel != "" {
				summary.ProductActivity.DeploymentModel = src.Metadata.DeploymentModel
			}
		case ActionNameComponentCreatedStart:
			componentCreatedStart = true
		case ActionNameComponentCreatedEnd:
			componentCreatedEnd = true
		case ActionNameQuickStartAttemptedSource:
			summary.ProductActivity.AttemptedSourceMethod = src.Metadata.Method
		case ActionNameQuickStartValidation:
			summary.ProductActivity.ValidationSource = src.Metadata.Source
			summary.ProductActivity.ValidationOutcome = src.Metadata.Outcome
		case ActionNameQuickStartSelectedSource:
			summary.ProductActivity.SelectedSource = src.Metadata.Source
		case ActionNameGatewayActivated:
			summary.ProductActivity.GatewayActivated = true
		case ActionNameComponentDeployed:
			summary.ProductActivity.ComponentDeployed = true
		case ActionNameComponentTested:
			summary.ProductActivity.ComponentTested = true
		case ActionNameComponentPromoted:
			summary.ProductActivity.ComponentPromoted = true
		case ActionNameComponentGeneratedKey:
			summary.ProductActivity.ComponentKeyGenerated = true
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

	summary.ProductActivity.QuickStartCompleted = !quickStartSkipped
	summary.ProductActivity.APICreated = componentCreatedStart && componentCreatedEnd

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

func parseAPIMTime(raw string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05.000", raw)
}
