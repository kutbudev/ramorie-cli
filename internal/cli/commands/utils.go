package commands

import (
	"github.com/AlecAivazis/survey/v2"
)

// Helper functions shared across commands

func stringPtr(s string) *string {
	return &s
}

// askForConfirmation prompts the user for a yes/no confirmation.
func askForConfirmation(prompt string) bool {
	confirmed := false
	confirmationPrompt := &survey.Confirm{
		Message: prompt,
		Default: false,
	}
	survey.AskOne(confirmationPrompt, &confirmed)
	return confirmed
}

// truncateString shortens a string to a specified length, adding "..." if truncated.
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
} 