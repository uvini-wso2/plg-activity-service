package moesif

import "testing"

func intPtr(i int) *int {
	return &i
}

func TestNormalize(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_456",
				ActionName: ActionNameOnboardingStepCompleted,
				Request:    RawRequest{Time: "2026-08-15T09:00:00.000"}, // earliest — FirstSeen
			},
		},
		{
			Source: RawSource{
				CompanyID:  "company_456",
				ActionName: ActionNameOrganizationCreated,
				Request:    RawRequest{Time: "2026-08-20T10:00:00.000"},
				Company:    RawCompany{Metadata: RawCompanyMetadata{AccountName: "test-org"}},
			},
		},
		{
			Source: RawSource{
				CompanyID:  "company_456",
				ActionName: ActionNameOnboardingSkipped,
				Request:    RawRequest{Time: "2026-08-20T10:15:00.000"},
				Metadata:   RawMetadata{StepNumber: intPtr(0), StepName: "welcome_option_selected"},
			},
		},
		{
			// The most recent skip — this one should win.
			Source: RawSource{
				CompanyID:  "company_456",
				ActionName: ActionNameOnboardingSkipped,
				Request:    RawRequest{Time: "2026-08-25T12:00:00.000"},
				Metadata:   RawMetadata{StepNumber: intPtr(3), StepName: "redirect_url_configured"},
			},
		},
		{
			Source: RawSource{
				CompanyID:  "company_456",
				ActionName: ActionNameOnboardingCompleted,
				Request:    RawRequest{Time: "2026-08-30T08:00:00.000"},
			},
		},
		{
			Source: RawSource{
				CompanyID:  "company_456",
				ActionName: ActionNameUserCreated,
				Request:    RawRequest{Time: "2026-08-31T08:20:00.000"}, // latest — LastActivity
			},
		},
	}

	summary := Normalize(hits)

	if summary.OrganizationName != "test-org" {
		t.Errorf("expected OrganizationName = test-org, got %q", summary.OrganizationName)
	}
	// Times are shown in Sri Lankan time (UTC+5:30) per team decision
	// (2026-09-16) — source times were UTC, so 09:00 -> 14:30 and
	// 08:20 -> 13:50.
	if summary.FirstSeen != "2026-08-15T14:30:00+05:30" {
		t.Errorf("expected FirstSeen = 2026-08-15T14:30:00+05:30 (Sri Lanka time), got %q", summary.FirstSeen)
	}
	if summary.LastActivity != "2026-08-31T13:50:00+05:30" {
		t.Errorf("expected LastActivity = 2026-08-31T13:50:00+05:30 (Sri Lanka time), got %q", summary.LastActivity)
	}

	if !summary.ProductActivity.ApplicationCreated {
		t.Error("expected ProductActivity.ApplicationCreated to be true")
	}
	if !summary.ProductActivity.HasCompletedOnboarding {
		t.Error("expected ProductActivity.HasCompletedOnboarding to be true (Onboarding-Completed event present)")
	}
	if summary.ProductActivity.SkippedStepNumber == nil {
		t.Fatal("expected SkippedStepNumber to be set, got nil")
	}
	if *summary.ProductActivity.SkippedStepNumber != 3 {
		t.Errorf("expected SkippedStepNumber = 3 (the chronologically latest skip), got %d", *summary.ProductActivity.SkippedStepNumber)
	}
	if summary.ProductActivity.SkippedStepName != "redirect_url_configured" {
		t.Errorf("expected SkippedStepName = redirect_url_configured, got %q", summary.ProductActivity.SkippedStepName)
	}
}

func TestNormalize_SingleEvent(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_456",
				ActionName: ActionNameOnboardingSkipped,
				Request:    RawRequest{Time: "2026-08-15T09:00:00.000"},
				Metadata:   RawMetadata{StepNumber: intPtr(1), StepName: "app_name_entered"},
			},
		},
	}

	summary := Normalize(hits)

	// 09:00 UTC -> 14:30 Sri Lanka time.
	if summary.FirstSeen != "2026-08-15T14:30:00+05:30" {
		t.Errorf("expected FirstSeen = 2026-08-15T14:30:00+05:30 (Sri Lanka time), got %q", summary.FirstSeen)
	}
	if summary.LastActivity != "2026-08-15T14:30:00+05:30" {
		t.Errorf("expected LastActivity = 2026-08-15T14:30:00+05:30 (Sri Lanka time), got %q", summary.LastActivity)
	}
	if summary.ProductActivity.SkippedStepNumber == nil {
		t.Error("expected SkippedStepNumber to be set")
	}
	if summary.ProductActivity.HasCompletedOnboarding {
		t.Error("expected HasCompletedOnboarding = false (no Onboarding-Completed event)")
	}
}

func TestNormalize_NoSkip(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_456",
				ActionName: ActionNameOnboardingStepCompleted,
				Request:    RawRequest{Time: "2026-08-15T09:00:00.000"},
			},
		},
	}

	summary := Normalize(hits)

	if summary.ProductActivity.SkippedStepNumber != nil {
		t.Errorf("expected SkippedStepNumber = nil (never skipped), got %v", *summary.ProductActivity.SkippedStepNumber)
	}
}

// TestNormalize_SkipAtStepZero confirms step 0 is correctly distinguished
// from "no skip" — a genuine skip at step 0 must show *0, not nil.
func TestNormalize_SkipAtStepZero(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_456",
				ActionName: ActionNameOnboardingSkipped,
				Request:    RawRequest{Time: "2026-08-15T09:00:00.000"},
				Metadata:   RawMetadata{StepNumber: intPtr(0), StepName: "welcome_option_selected"},
			},
		},
	}

	summary := Normalize(hits)

	if summary.ProductActivity.SkippedStepNumber == nil {
		t.Fatal("expected SkippedStepNumber to be set (step 0 is a real skip), got nil")
	}
	if *summary.ProductActivity.SkippedStepNumber != 0 {
		t.Errorf("expected SkippedStepNumber = 0, got %d", *summary.ProductActivity.SkippedStepNumber)
	}
}

func TestNormalize_NoOrganizationName(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_456",
				ActionName: ActionNameUserCreated,
				Request:    RawRequest{Time: "2026-08-15T09:00:00.000"},
			},
		},
	}

	summary := Normalize(hits)

	if summary.OrganizationName != "" {
		t.Errorf("expected OrganizationName = \"\" (no event carried it), got %q", summary.OrganizationName)
	}
}

func TestNormalize_GeoAndWizardPath(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_456",
				ActionName: ActionNameOnboardingStepCompleted,
				Request: RawRequest{
					Time:  "2026-08-15T09:00:00.000",
					GeoIP: RawGeoIP{Timezone: "Asia/Colombo", CountryName: "Sri Lanka"},
				},
				Metadata: RawMetadata{WizardPath: "full_setup"},
			},
		},
		{
			Source: RawSource{
				CompanyID:  "company_456",
				ActionName: ActionNameOnboardingStepCompleted,
				Request: RawRequest{
					Time:  "2026-08-20T10:00:00.000",
					GeoIP: RawGeoIP{Timezone: "America/New_York", CountryName: "United States"},
				},
			},
		},
	}

	summary := Normalize(hits)

	if summary.Timezone != "America/New_York" {
		t.Errorf("expected Timezone = America/New_York (from the MOST RECENT event), got %q", summary.Timezone)
	}
	if summary.CountryName != "United States" {
		t.Errorf("expected CountryName = United States (from the MOST RECENT event), got %q", summary.CountryName)
	}
	if summary.ProductActivity.OnboardingSetupType != "full_setup" {
		t.Errorf("expected OnboardingSetupType = full_setup, got %q", summary.ProductActivity.OnboardingSetupType)
	}
}

func TestNormalize_NoGeoData(t *testing.T) {
	hits := []RawHit{
		{
			Source: RawSource{
				CompanyID:  "company_456",
				ActionName: ActionNameUserCreated,
				Request:    RawRequest{Time: "2026-08-15T09:00:00.000"},
			},
		},
	}

	summary := Normalize(hits)

	if summary.Timezone != "" {
		t.Errorf("expected Timezone = \"\" (no geo data present), got %q", summary.Timezone)
	}
	if summary.CountryName != "" {
		t.Errorf("expected CountryName = \"\" (no geo data present), got %q", summary.CountryName)
	}
	if summary.ProductActivity.OnboardingSetupType != "" {
		t.Errorf("expected OnboardingSetupType = \"\" (no wizard_path present), got %q", summary.ProductActivity.OnboardingSetupType)
	}
}
