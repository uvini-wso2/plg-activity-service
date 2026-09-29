package email

import (
	"fmt"
	"strings"

	"github.com/uvini-wso2/plg-activity-service/internal/apim"
)

// SummarizeAPIMActivity turns an apim.Summary into a short, plain-text
// description suitable for feeding to Claude — mirrors SummarizeActivity
// (the Asgardeo version), but describes APIM's own product-specific
// signals instead.
func SummarizeAPIMActivity(s apim.Summary) string {
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

	pa := s.ProductActivity

	if pa.QuickStartCompleted {
		lines = append(lines, "Completed the quick-start guide.")
	} else {
		lines = append(lines, "Did not complete the quick-start guide (skipped partway).")
	}

	if pa.QuickStartSelectedProduct {
		if pa.DeploymentModel != "" {
			lines = append(lines, fmt.Sprintf("Selected a product to try, using the %s deployment option.", pa.DeploymentModel))
		} else {
			lines = append(lines, "Selected a product to try during quick-start.")
		}
	}

	if pa.APICreated {
		lines = append(lines, "Created a real API/component.")
	}
	if pa.GatewayActivated {
		lines = append(lines, "Activated the API gateway.")
	}
	if pa.ComponentDeployed {
		lines = append(lines, "Deployed their component — it's live.")
	}
	if pa.ComponentTested {
		lines = append(lines, "Tested their component.")
	}
	if pa.ComponentPromoted {
		lines = append(lines, "Promoted their component to the next stage.")
	}
	if pa.ComponentKeyGenerated {
		lines = append(lines, "Generated an API access key.")
	}

	if len(lines) == 0 {
		return "No activity data available."
	}
	return strings.Join(lines, "\n")
}
