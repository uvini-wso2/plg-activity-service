package apim

import (
	"testing"

	"github.com/uvini-wso2/plg-activity-service/internal/validation"
)

func TestClassify_Disposable(t *testing.T) {
	ec := validation.EmailClassification{Domain: "mailinator.com", Category: validation.CategoryDisposable}
	result := Classify(ec, Summary{})

	if result.Outcome != validation.OutcomeExcluded {
		t.Errorf("expected Excluded, got %s", result.Outcome)
	}
}

func TestClassify_ProviderTestingExcluded(t *testing.T) {
	ec := validation.EmailClassification{Domain: "gmail.com", Category: validation.CategoryProviderTesting}
	result := Classify(ec, Summary{ProductActivity: ProductActivity{HasMeaningfulActivity: true}})

	if result.Outcome != validation.OutcomeExcluded {
		t.Errorf("expected Excluded regardless of activity, got %s", result.Outcome)
	}
}

func TestClassify_WSO2DomainExcluded(t *testing.T) {
	ec := validation.EmailClassification{Domain: "wso2.com", Category: validation.CategoryCorporate}
	result := Classify(ec, Summary{})

	if result.Outcome != validation.OutcomeExcluded {
		t.Errorf("expected Excluded, got %s", result.Outcome)
	}
	if len(result.Tags) != 1 || result.Tags[0] != validation.TagWSO2Domain {
		t.Errorf("expected tags [%s], got %v", validation.TagWSO2Domain, result.Tags)
	}
}

func TestClassify_WSO2BeatsCorporate(t *testing.T) {
	ec := validation.EmailClassification{Domain: "wso2.com", Category: validation.CategoryCorporate}
	summary := Summary{ProductActivity: ProductActivity{HasMeaningfulActivity: true}}
	result := Classify(ec, summary)

	if result.Outcome != validation.OutcomeExcluded {
		t.Fatalf("expected WSO2 domain to be Excluded regardless of category, got %s", result.Outcome)
	}
}

func TestClassify_Corporate(t *testing.T) {
	ec := validation.EmailClassification{Domain: "acme.com", Category: validation.CategoryCorporate}
	result := Classify(ec, Summary{})

	if result.Outcome != validation.OutcomeEligible {
		t.Errorf("expected Eligible, got %s", result.Outcome)
	}
}

func TestClassify_PersonalWithMeaningfulActivity(t *testing.T) {
	ec := validation.EmailClassification{Domain: "gmail.com", Category: validation.CategoryPersonal}
	summary := Summary{ProductActivity: ProductActivity{HasMeaningfulActivity: true}}
	result := Classify(ec, summary)

	if result.Outcome != validation.OutcomeEligible {
		t.Errorf("expected Eligible, got %s", result.Outcome)
	}
}

func TestClassify_PersonalWithNoActivity(t *testing.T) {
	ec := validation.EmailClassification{Domain: "gmail.com", Category: validation.CategoryPersonal}
	result := Classify(ec, Summary{})

	if result.Outcome != validation.OutcomeMonitored {
		t.Errorf("expected Monitored, got %s", result.Outcome)
	}
}
