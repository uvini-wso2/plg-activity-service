package handler

import "github.com/uvini-wso2/plg-activity-service/internal/classification"

// classificationClient is the interface Validate() and APIMValidate()
// depend on for real email classification (2026-09-30) — replacing the
// earlier temporary approach of accepting domain/category as raw query
// params.
type classificationClient interface {
	Classify(email string) (classification.Response, error)
}
