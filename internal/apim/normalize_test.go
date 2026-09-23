package apim

import "testing"

func TestNormalize_BasicFields(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_789",
				ActionName: ActionNameLandingSignInSucceeded,
				Request: RawRequest{
					Time:  "2026-09-14T08:30:00.000",
					GeoIP: RawGeoIP{Timezone: "Asia/Colombo", CountryName: "Sri Lanka"},
				},
				Company: RawCompany{Metadata: RawCompanyMetadata{Name: "shopwaveorg"}},
				User:    RawUser{Metadata: RawUserMetadata{IsWSO2User: "false"}},
			},
		},
	}

	summary := Normalize(hits, 3)

	if summary.OrganizationName != "shopwaveorg" {
		t.Errorf("expected OrganizationName = shopwaveorg, got %q", summary.OrganizationName)
	}
	if summary.IsWSO2User {
		t.Error("expected IsWSO2User = false")
	}
	if summary.EventsFound != 3 {
		t.Errorf("expected EventsFound = 3 (the real Moesif total, not len(hits)), got %d", summary.EventsFound)
	}
	// 08:30 UTC -> 2:00 PM Sri Lanka time.
	if summary.LastActivity != "September 14, 2026 2:00 PM" {
		t.Errorf("expected LastActivity = September 14, 2026 2:00 PM, got %q", summary.LastActivity)
	}
}

func TestNormalize_IsWSO2User(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_789",
				ActionName: ActionNameLandingSignInSucceeded,
				Request:    RawRequest{Time: "2026-09-14T08:30:00.000"},
				User:       RawUser{Metadata: RawUserMetadata{IsWSO2User: "true"}},
			},
		},
	}

	summary := Normalize(hits, 1)

	if !summary.IsWSO2User {
		t.Error("expected IsWSO2User = true when isWSO2User field is the string \"true\"")
	}
}

func TestNormalize_MeaningfulActivity_AboveThreshold(t *testing.T) {
	summary := Normalize([]RawHit{}, 400)

	if !summary.ProductActivity.HasMeaningfulActivity {
		t.Error("expected HasMeaningfulActivity = true at exactly 400 (threshold is inclusive)")
	}
}

func TestNormalize_MeaningfulActivity_BelowThreshold(t *testing.T) {
	summary := Normalize([]RawHit{}, 399)

	if summary.ProductActivity.HasMeaningfulActivity {
		t.Error("expected HasMeaningfulActivity = false at 399 (below threshold)")
	}
}

func TestNormalize_NoOrganizationName(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_789",
				ActionName: ActionNameLandingSignInSucceeded,
				Request:    RawRequest{Time: "2026-09-14T08:30:00.000"},
			},
		},
	}

	summary := Normalize(hits, 1)

	if summary.OrganizationName != "" {
		t.Errorf("expected OrganizationName = \"\" (no event carried it), got %q", summary.OrganizationName)
	}
}

func TestNormalize_NoEvents(t *testing.T) {
	summary := Normalize([]RawHit{}, 0)

	if summary.FirstSeen != "" || summary.LastActivity != "" {
		t.Error("expected empty FirstSeen/LastActivity when there are no events at all")
	}
	if summary.EventsFound != 0 {
		t.Errorf("expected EventsFound = 0, got %d", summary.EventsFound)
	}
}

func TestNormalize_QuickStartSelectedProduct(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_789",
				ActionName: ActionNameQuickStartSelectedProduct,
				Request:    RawRequest{Time: "2026-09-14T08:30:00.000"},
				Metadata:   RawMetadata{DeploymentModel: "gateway"},
			},
		},
	}

	summary := Normalize(hits, 1)

	if !summary.ProductActivity.QuickStartSelectedProduct {
		t.Error("expected ProductActivity.QuickStartSelectedProduct = true")
	}
	if summary.ProductActivity.DeploymentModel != "gateway" {
		t.Errorf("expected DeploymentModel = gateway, got %q", summary.ProductActivity.DeploymentModel)
	}
}

// TestNormalize_QuickStartCompleted_NoSkip confirms the renamed + inverted
// logic (2026-09-22): QuickStartCompleted is true when NO skip event
// exists at all.
func TestNormalize_QuickStartCompleted_NoSkip(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_789",
				ActionName: ActionNameLandingSignInSucceeded,
				Request:    RawRequest{Time: "2026-09-14T08:30:00.000"},
			},
		},
	}

	summary := Normalize(hits, 1)

	if !summary.ProductActivity.QuickStartCompleted {
		t.Error("expected QuickStartCompleted = true when no skip event is present")
	}
}

// TestNormalize_QuickStartCompleted_WithSkip confirms the inverted case:
// a real skip event present means QuickStartCompleted = false.
func TestNormalize_QuickStartCompleted_WithSkip(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_789",
				ActionName: ActionNameQuickStartSkipped,
				Request:    RawRequest{Time: "2026-09-14T08:30:00.000"},
			},
		},
	}

	summary := Normalize(hits, 1)

	if summary.ProductActivity.QuickStartCompleted {
		t.Error("expected QuickStartCompleted = false when a skip event is present")
	}
}

// TestNormalize_APICreated_BothEventsPresent confirms APICreated requires
// BOTH Component-Created-Start AND Component-Created-End (2026-09-22).
func TestNormalize_APICreated_BothEventsPresent(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_789",
				ActionName: ActionNameComponentCreatedStart,
				Request:    RawRequest{Time: "2026-09-14T08:30:00.000"},
			},
		},
		{
			Source: RawSource{
				CompanyID:  "company_789",
				ActionName: ActionNameComponentCreatedEnd,
				Request:    RawRequest{Time: "2026-09-14T08:31:00.000"},
			},
		},
	}

	summary := Normalize(hits, 2)

	if !summary.ProductActivity.APICreated {
		t.Error("expected APICreated = true when BOTH start and end events are present")
	}
}

// TestNormalize_APICreated_OnlyStartPresent confirms APICreated stays
// false if only the Start event exists, without a matching End.
func TestNormalize_APICreated_OnlyStartPresent(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_789",
				ActionName: ActionNameComponentCreatedStart,
				Request:    RawRequest{Time: "2026-09-14T08:30:00.000"},
			},
		},
	}

	summary := Normalize(hits, 1)

	if summary.ProductActivity.APICreated {
		t.Error("expected APICreated = false when only Component-Created-Start is present, without End")
	}
}

func TestNormalize_QuickStartFunnelMetadata(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_789",
				ActionName: ActionNameQuickStartAttemptedSource,
				Request:    RawRequest{Time: "2026-09-14T08:30:00.000"},
				Metadata:   RawMetadata{Method: "import"},
			},
		},
		{
			Source: RawSource{
				CompanyID:  "company_789",
				ActionName: ActionNameQuickStartValidation,
				Request:    RawRequest{Time: "2026-09-14T08:31:00.000"},
				Metadata:   RawMetadata{Source: "github", Outcome: "success"},
			},
		},
		{
			Source: RawSource{
				CompanyID:  "company_789",
				ActionName: ActionNameQuickStartSelectedSource,
				Request:    RawRequest{Time: "2026-09-14T08:32:00.000"},
				Metadata:   RawMetadata{Source: "github"},
			},
		},
	}

	summary := Normalize(hits, 3)

	if summary.ProductActivity.AttemptedSourceMethod != "import" {
		t.Errorf("expected AttemptedSourceMethod = import, got %q", summary.ProductActivity.AttemptedSourceMethod)
	}
	if summary.ProductActivity.ValidationSource != "github" {
		t.Errorf("expected ValidationSource = github, got %q", summary.ProductActivity.ValidationSource)
	}
	if summary.ProductActivity.ValidationOutcome != "success" {
		t.Errorf("expected ValidationOutcome = success, got %q", summary.ProductActivity.ValidationOutcome)
	}
	if summary.ProductActivity.SelectedSource != "github" {
		t.Errorf("expected SelectedSource = github, got %q", summary.ProductActivity.SelectedSource)
	}
}

func TestNormalize_ComponentLifecycleFlags(t *testing.T) {
	hits := []RawHit{
		{Source: RawSource{ActionName: ActionNameGatewayActivated, Request: RawRequest{Time: "2026-09-14T08:30:00.000"}}},
		{Source: RawSource{ActionName: ActionNameComponentDeployed, Request: RawRequest{Time: "2026-09-14T08:31:00.000"}}},
		{Source: RawSource{ActionName: ActionNameComponentTested, Request: RawRequest{Time: "2026-09-14T08:32:00.000"}}},
		{Source: RawSource{ActionName: ActionNameComponentPromoted, Request: RawRequest{Time: "2026-09-14T08:33:00.000"}}},
		{Source: RawSource{ActionName: ActionNameComponentGeneratedKey, Request: RawRequest{Time: "2026-09-14T08:34:00.000"}}},
	}

	summary := Normalize(hits, 5)

	if !summary.ProductActivity.GatewayActivated {
		t.Error("expected GatewayActivated = true")
	}
	if !summary.ProductActivity.ComponentDeployed {
		t.Error("expected ComponentDeployed = true")
	}
	if !summary.ProductActivity.ComponentTested {
		t.Error("expected ComponentTested = true")
	}
	if !summary.ProductActivity.ComponentPromoted {
		t.Error("expected ComponentPromoted = true")
	}
	if !summary.ProductActivity.ComponentKeyGenerated {
		t.Error("expected ComponentKeyGenerated = true")
	}
}
