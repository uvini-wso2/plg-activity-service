package email

import (
	"fmt"
	"strings"

	"github.com/uvini-wso2/plg-activity-service/internal/moesif"
)

// SummarizeActivity turns a moesif.Summary into a short, plain-text
// description suitable for feeding to Claude — deliberately only
// describing what we actually know, per the "don't invent info" rule.
func SummarizeActivity(s moesif.Summary) string {
	var lines []string

	if s.OrganizationName != "" {
		lines = append(lines, fmt.Sprintf("Organization: %s", s.OrganizationName))
	}
	if s.FirstSeen != "" {
		lines = append(lines, fmt.Sprintf("First seen: %s", s.FirstSeen))
	}
	if s.LastActivity != "" {
		lines = append(lines, fmt.Sprintf("Last active: %s", s.LastActivity))
	}

	if s.ProductActivity.ApplicationCreated {
		lines = append(lines, "Completed onboarding and created an application.")
	} else if s.ProductActivity.HasSkippedOnboarding {
		step := "an early step"
		if s.ProductActivity.SkippedStepName != "" {
			step = s.ProductActivity.SkippedStepName
		}
		lines = append(lines, fmt.Sprintf("Started onboarding but stopped at: %s.", step))
	} else {
		lines = append(lines, "No onboarding activity recorded yet.")
	}

	if len(lines) == 0 {
		return "No activity data available."
	}
	return strings.Join(lines, "\n")
}
