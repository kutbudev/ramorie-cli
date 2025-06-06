package commands

// Helper functions shared across commands

func stringPtr(s string) *string {
	return &s
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
} 