package validation

import "github.com/uvini-wso2/plg-activity-service/internal/moesif"

// Classify combines an email classification result with Moesif product
// activity to produce a three-way PLG CS outcome, per rules given by the
// team (2026-09-11).
//
// Order matters: exclusion checks (disposable, WSO2 domain) run BEFORE
// the corporate-eligibility check. This is deliberate — a @wso2.com email
// would likely also be categorized "corporate" by the classification API,
// so WSO2 must be excluded first or it would incorrectly qualify as
// Eligible via the corporate rule.
func Classify(ec EmailClassification, summary moesif.Summary) Result {
	// Disposable emails are excluded unconditionally, regardless of product
	// activity — even a disposable-email account with lots of real
	// engagement isn't a genuine, reachable customer, so there's no
	// meaningful-activity check here at all (confirmed with team,
	// 2026-09-11).
	if ec.Category == CategoryDisposable {
		return Result{Outcome: OutcomeExcluded, Tags: []string{TagDisposableDomain}}
	}

	if isWSO2Domain(ec.Domain) {
		return Result{Outcome: OutcomeExcluded, Tags: []string{TagWSO2Domain}}
	}

	//provider_testing (e.g. test@gmail.com, role-based/testing mailboxes)
	// is confirmed invalid — excluded outright, not checked for meaningful
	// activity like personal email (confirmed with team, 2026-09-13).
	if ec.Category == CategoryProviderTesting {
		return Result{Outcome: OutcomeExcluded, Tags: []string{TagInvalidEmail}}
	}

	if ec.Category == CategoryCorporate {
		// Corporate is Eligible regardless of product activity, per team rule.
		return Result{Outcome: OutcomeEligible, Tags: []string{TagCorporateDomain}}
	}

	// Generic/free email (personal, or provider_testing treated the same
	// way per team decision 2026-09-11) qualifies as Eligible only with
	// meaningful product activity.
	if ec.Category == CategoryPersonal && hasMeaningfulActivity(summary) {
		return Result{Outcome: OutcomeEligible, Tags: []string{TagMeaningfulProductActivity}}
	}

	return Result{Outcome: OutcomeMonitored, Tags: []string{}}
}

// hasMeaningfulActivity defines "meaningful product activity" as the
// account having done enough that a CS engineer has something SPECIFIC to
// reference and offer help with — not just a bare signup with nothing to
// point to.
//
// Two situations both count, since both give CS a concrete talking point:
//   - ApplicationCreated: they completed onboarding and built something
//     ("saw you set up your app!")
//   - HasSkippedOnboarding: they engaged with onboarding but stopped at an
//     identifiable point ("saw you paused at <step>, need a hand?")
//
// PROPOSAL, NOT YET CONFIRMED (2026-09-11) — this definition was not given
// explicitly by the team; it's our best interpretation using currently
// available Moesif fields. Confirm with Supeshala before relying on this
// for real classification decisions.
func hasMeaningfulActivity(summary moesif.Summary) bool {
	return summary.ProductActivity.ApplicationCreated || summary.ProductActivity.SkippedStepNumber != nil
}
