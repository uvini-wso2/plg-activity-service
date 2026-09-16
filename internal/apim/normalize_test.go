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
		{
			Source: RawSource{
				CompanyID:  "company_789",
				ActionName: ActionNameProjectCreatedStart,
				Request: RawRequest{
					Time:  "2026-09-14T09:00:00.000",
					GeoIP: RawGeoIP{Timezone: "Asia/Colombo", CountryName: "Sri Lanka"},
				},
			},
		},
	}

	summary := Normalize(hits, 3)

	if summary.OrganizationName != "shopwaveorg" {
		t.Errorf("expected OrganizationName = shopwaveorg, got %q", summary.OrganizationName)
	}
	if !summary.ProductActivity.SignedIn {
		t.Error("expected ProductActivity.SignedIn = true")
	}
	if !summary.ProductActivity.ProjectCreated {
		t.Error("expected ProductActivity.ProjectCreated = true")
	}
	if summary.ProductActivity.ComponentCreated {
		t.Error("expected ProductActivity.ComponentCreated = false (no such event present)")
	}
	if summary.IsWSO2User {
		t.Error("expected IsWSO2User = false")
	}
	if summary.EventsFound != 3 {
		t.Errorf("expected EventsFound = 3 (the real Moesif total, not len(hits)), got %d", summary.EventsFound)
	}
	// 09:00 UTC -> 14:30 Sri Lanka time.
	if summary.LastActivity != "2026-09-14T14:30:00+05:30" {
		t.Errorf("expected LastActivity = 2026-09-14T14:30:00+05:30, got %q", summary.LastActivity)
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
	summary := Normalize([]RawHit{}, 500)

	if !summary.ProductActivity.HasMeaningfulActivity {
		t.Error("expected HasMeaningfulActivity = true at exactly 500 (threshold is inclusive)")
	}
}

func TestNormalize_MeaningfulActivity_BelowThreshold(t *testing.T) {
	summary := Normalize([]RawHit{}, 499)

	if summary.ProductActivity.HasMeaningfulActivity {
		t.Error("expected HasMeaningfulActivity = false at 499 (below threshold)")
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
