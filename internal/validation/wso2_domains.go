package validation

import "strings"

// wso2Domains lists domains treated as internal WSO2 domains for exclusion
// purposes. ASSUMPTION (2026-09-11): only wso2.com confirmed so far — add
// any other real internal domains here once confirmed (e.g. regional
// variants), rather than guessing more.
var wso2Domains = []string{
	"wso2.com",
}

func IsWSO2Domain(domain string) bool {
	domain = strings.ToLower(domain)
	for _, d := range wso2Domains {
		if domain == d {
			return true
		}
	}
	return false
}
