package apim

import "time"

// Action name values for APIM's "action_name" field — CONFIRMED real
// values, expanded per team decisions (2026-09-22, 2026-09-24).
const (
	ActionNameLandingSignInSucceeded    = "Landing-SignIn-Succeeded"
	ActionNameQuickStartSkipped         = "QuickStart-Skipped"
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

	// ActionNameAPIInvoked: CONFIRMED real (seen on the apimsaas company,
	// 2026-09-21). Now wired in per team decision (2026-09-24) — used to
	// redefine the meaningful-activity threshold, counting ONLY these
	// events instead of raw eventsFound.
	ActionNameAPIInvoked = "API-Invoked"

	// Confirmed real, but explicitly NOT tracked per team decision
	// (2026-09-22) — kept as reference only:
	//   ActionNameLandingSignUpSucceeded = "Landing-SignUp-Succeeded"
	//   ActionNamePortalViewedHome        = "Portal-Viewed-Home"
	//   ActionNameHomePageVisit           = "home-page-visit"
	//   ActionNameProjectCreatedStart     = "Project-Created-Start"
	//   ActionNameLandingSignInViewed     = "Landing-SignIn-Viewed"
	//   ActionNameLandingSignInFailed     = "Landing-SignIn-Failed"
	//   ActionNameLandingViewedPage       = "Landing-Viewed-Page"
)

var sriLankaLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Colombo")
	if err != nil {
		return time.UTC
	}
	return loc
}()

const timeOutputLayout = "January 2, 2006 3:04 PM"

// ProductActivity holds signals specific to APIM.
type ProductActivity struct {
	QuickStartCompleted bool `json:"quickStartCompleted"`

	QuickStartSelectedProduct bool   `json:"quickStartSelectedProduct"`
	DeploymentModel           string `json:"deploymentModel,omitempty"`
	// Context: NEW (2026-09-24) — a separate metadata value on
	// QuickStart-Selected-Product, alongside DeploymentModel.
	Context string `json:"context,omitempty"`

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

	// APIInvokedCount / HasMeaningfulActivity: REDEFINED (2026-09-24) —
	// counts specifically how many API-Invoked events occurred (within
	// the fetched event window — see Search()'s pagination cap), rather
	// than raw eventsFound. This directly addresses the confirmed
	// duplicate-event/bot-traffic noise found in raw counts.
	APIInvokedCount       int  `json:"apiInvokedCount"`
	HasMeaningfulActivity bool `json:"hasMeaningfulActivity"`
}

// Summary: IsWSO2User REMOVED (2026-09-24) — APIM now uses the same
// domain-based WSO2 check as every other product (see
// internal/validation.IsWSO2Domain), not its own direct signal.
type Summary struct {
	OrganizationName string          `json:"organizationName,omitempty"`
	FirstSeen        string          `json:"firstSeen"`
	LastActivity     string          `json:"lastActivity"`
	Timezone         string          `json:"timezone,omitempty"`
	CountryName      string          `json:"countryName,omitempty"`
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
	var apiInvokedCount int

	for _, hit := range hits {
		src := hit.Source

		if summary.OrganizationName == "" && src.Company.Metadata.Name != "" {
			summary.OrganizationName = src.Company.Metadata.Name
		}

		switch src.ActionName {
		case ActionNameQuickStartSkipped:
			quickStartSkipped = true
		case ActionNameQuickStartSelectedProduct:
			summary.ProductActivity.QuickStartSelectedProduct = true
			if src.Metadata.DeploymentModel != "" {
				summary.ProductActivity.DeploymentModel = src.Metadata.DeploymentModel
			}
			if src.Metadata.Context != "" {
				summary.ProductActivity.Context = src.Metadata.Context
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
		case ActionNameAPIInvoked:
			apiInvokedCount++
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
	summary.ProductActivity.APIInvokedCount = apiInvokedCount
	summary.ProductActivity.HasMeaningfulActivity = apiInvokedCount >= meaningfulActivityThreshold

	return summary
}

func parseAPIMTime(raw string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05.000", raw)
}
