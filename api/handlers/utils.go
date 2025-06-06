package handlers

import "github.com/terzigolu/josepshbrain-go/api/models"

// NormalizePriority converts descriptive priority strings to their single-character representation.
// It ensures that the API can accept user-friendly priority names while storing them
// in the database-compliant single-character format.
func NormalizePriority(p string) models.Priority {
	switch p {
	case "HIGH", "H":
		return models.PriorityHigh
	case "MEDIUM", "M":
		return models.PriorityMedium
	case "LOW", "L":
		return models.PriorityLow
	default:
		return models.PriorityMedium // Default to Medium if unknown
	}
} 