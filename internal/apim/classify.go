package apim

import "github.com/uvini-wso2/plg-activity-service/internal/validation"

// Classify applies the same outcome/tag structure used for Asgardeo
// (internal/validation.Classify), but uses APIM's own meaningful-activity
// signal.
//
// PROPOSAL, NOT FULLY CONFIRMED (2026-09-22): the team confirmed
// "meaningful activity requirements are different" for APIM, but has not
// explicitly confirmed whether the OTHER rules (corporate/disposable/WSO2)
// should also differ. This assumes only the meaningful-activity check
// itself is product-specific, reusing everything else. Confirm with the
// team before relying on this for real decisions.
func Classify(ec validation.EmailClassification, summary Summary) validation.Result {
	if ec.Category == validation.CategoryDisposable {
		return validation.Result{Outcome: validation.OutcomeExcluded, Tags: []string{validation.TagDisposableDomain}}
	}

	if ec.Category == validation.CategoryProviderTesting {
		return validation.Result{Outcome: validation.OutcomeExcluded, Tags: []string{validation.TagInvalidEmail}}
	}

	// isWSO2User check REMOVED (2026-09-24) — APIM now uses the same
	// domain-based check as every other product, for consistency.
	if validation.IsWSO2Domain(ec.Domain) {
		return validation.Result{Outcome: validation.OutcomeExcluded, Tags: []string{validation.TagWSO2Domain}}
	}

	if ec.Category == validation.CategoryCorporate {
		return validation.Result{Outcome: validation.OutcomeEligible, Tags: []string{validation.TagCorporateDomain}}
	}

	if ec.Category == validation.CategoryPersonal && summary.ProductActivity.HasMeaningfulActivity {
		return validation.Result{Outcome: validation.OutcomeEligible, Tags: []string{validation.TagMeaningfulProductActivity}}
	}

	return validation.Result{Outcome: validation.OutcomeMonitored, Tags: []string{}}
}
