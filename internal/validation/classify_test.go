package validation

import (
	"testing"

	"github.com/uvini-wso2/plg-activity-service/internal/moesif"
)

func TestClassify_Disposable(t *testing.T) {
	ec := EmailClassification{Domain: "mailinator.com", Category: CategoryDisposable}
	result := Classify(ec, moesif.Summary{})

	if result.Outcome != OutcomeExcluded {
		t.Errorf("expected Excluded, got %s", result.Outcome)
	}
	if len(result.Tags) != 1 || result.Tags[0] != TagDisposableDomain {
		t.Errorf("expected tags [%s], got %v", TagDisposableDomain, result.Tags)
	}
}

func TestClassify_WSO2Domain(t *testing.T) {
	ec := EmailClassification{Domain: "wso2.com", Category: CategoryCorporate}
	result := Classify(ec, moesif.Summary{})

	if result.Outcome != OutcomeExcluded {
		t.Errorf("expected Excluded, got %s", result.Outcome)
	}
	if len(result.Tags) != 1 || result.Tags[0] != TagWSO2Domain {
		t.Errorf("expected tags [%s], got %v", TagWSO2Domain, result.Tags)
	}
}

// TestClassify_WSO2DomainBeatsCorporate specifically confirms the ordering
// requirement: a wso2.com email categorized "corporate" by the
// classification API must still be Excluded, not Eligible.
func TestClassify_WSO2DomainBeatsCorporate(t *testing.T) {
	ec := EmailClassification{Domain: "wso2.com", Category: CategoryCorporate}
	result := Classify(ec, moesif.Summary{ProductActivity: moesif.ProductActivity{ApplicationCreated: true}})

	if result.Outcome != OutcomeExcluded {
		t.Fatalf("expected WSO2 domain to be Excluded regardless of category/activity, got %s", result.Outcome)
	}
}

func TestClassify_Corporate(t *testing.T) {
	ec := EmailClassification{Domain: "acme.com", Category: CategoryCorporate}
	// No product activity at all — should still be Eligible per team rule
	// ("if corporate, definitely regardless of product activities").
	result := Classify(ec, moesif.Summary{})

	if result.Outcome != OutcomeEligible {
		t.Errorf("expected Eligible, got %s", result.Outcome)
	}
	if len(result.Tags) != 1 || result.Tags[0] != TagCorporateDomain {
		t.Errorf("expected tags [%s], got %v", TagCorporateDomain, result.Tags)
	}
}

func TestClassify_PersonalWithMeaningfulActivity_ApplicationCreated(t *testing.T) {
	ec := EmailClassification{Domain: "gmail.com", Category: CategoryPersonal}
	summary := moesif.Summary{ProductActivity: moesif.ProductActivity{ApplicationCreated: true}}
	result := Classify(ec, summary)

	if result.Outcome != OutcomeEligible {
		t.Errorf("expected Eligible, got %s", result.Outcome)
	}
	if len(result.Tags) != 1 || result.Tags[0] != TagMeaningfulProductActivity {
		t.Errorf("expected tags [%s], got %v", TagMeaningfulProductActivity, result.Tags)
	}
}

// TestClassify_PersonalWithMeaningfulActivity_SkippedOnboarding confirms a
// stalled-but-identifiable onboarding attempt also counts as meaningful,
// per the clarified definition (CS has something specific to reference).
func TestClassify_PersonalWithMeaningfulActivity_SkippedOnboarding(t *testing.T) {
	ec := EmailClassification{Domain: "gmail.com", Category: CategoryPersonal}
	stepZero := 0
	summary := moesif.Summary{ProductActivity: moesif.ProductActivity{SkippedStepNumber: &stepZero}}
	result := Classify(ec, summary)

	if result.Outcome != OutcomeEligible {
		t.Errorf("expected Eligible (skipped onboarding still counts as meaningful), got %s", result.Outcome)
	}
}

func TestClassify_PersonalWithNoActivity(t *testing.T) {
	ec := EmailClassification{Domain: "gmail.com", Category: CategoryPersonal}
	result := Classify(ec, moesif.Summary{}) // no activity at all

	if result.Outcome != OutcomeMonitored {
		t.Errorf("expected Monitored, got %s", result.Outcome)
	}
}

// TestClassify_ProviderTestingExcluded confirms the corrected rule
// (2026-09-13): provider_testing emails are invalid and excluded
// outright, regardless of activity — NOT treated like personal email.
func TestClassify_ProviderTestingExcluded(t *testing.T) {
	ec := EmailClassification{Domain: "gmail.com", Category: CategoryProviderTesting}
	summary := moesif.Summary{ProductActivity: moesif.ProductActivity{ApplicationCreated: true}}
	result := Classify(ec, summary)

	if result.Outcome != OutcomeExcluded {
		t.Errorf("expected Excluded (provider_testing is invalid, regardless of activity), got %s", result.Outcome)
	}
	if len(result.Tags) != 1 || result.Tags[0] != TagInvalidEmail {
		t.Errorf("expected tags [%s], got %v", TagInvalidEmail, result.Tags)
	}
}
