package mcp

import (
	"regexp"
	"strings"
)

// Memory type constants
const (
	MemoryTypeGeneral    = "general"
	MemoryTypeDecision   = "decision"
	MemoryTypeBugFix     = "bug_fix"
	MemoryTypePreference = "preference"
	MemoryTypePattern    = "pattern"
	MemoryTypeReference  = "reference"
)

// DetectMemoryType analyzes content and returns the most appropriate memory type
func DetectMemoryType(content string) string {
	lowerContent := strings.ToLower(content)

	// Check for decision indicators
	decisionKeywords := []string{
		"decided", "chose", "chosen", "selected", "instead of",
		"opted for", "went with", "decision:", "adr", "architectural decision",
		"we will use", "we chose", "after considering",
	}
	for _, kw := range decisionKeywords {
		if strings.Contains(lowerContent, kw) {
			return MemoryTypeDecision
		}
	}

	// Check for bug fix indicators
	bugFixKeywords := []string{
		"fixed", "solved", "resolved", "bug:", "error:", "issue:",
		"root cause", "the problem was", "fix:", "hotfix",
		"patched", "workaround", "debugging",
	}
	for _, kw := range bugFixKeywords {
		if strings.Contains(lowerContent, kw) {
			return MemoryTypeBugFix
		}
	}

	// Check for preference indicators
	preferenceKeywords := []string{
		"prefer", "always use", "never use", "i like", "we like",
		"recommended", "best practice is", "should always", "should never",
		"convention:", "standard:", "rule:",
	}
	for _, kw := range preferenceKeywords {
		if strings.Contains(lowerContent, kw) {
			return MemoryTypePreference
		}
	}

	// Check for pattern indicators
	patternKeywords := []string{
		"pattern:", "design pattern", "architecture pattern",
		"implementation pattern", "coding pattern", "best practice",
		"template:", "boilerplate", "structure:",
	}
	for _, kw := range patternKeywords {
		if strings.Contains(lowerContent, kw) {
			return MemoryTypePattern
		}
	}

	// Check for reference indicators (URLs, documentation links)
	urlPattern := regexp.MustCompile(`https?://[^\s]+`)
	if urlPattern.MatchString(content) {
		// Check if it's primarily a reference
		referenceKeywords := []string{
			"documentation", "docs", "reference", "link:", "see:", "url:",
			"source:", "article", "tutorial", "guide",
		}
		for _, kw := range referenceKeywords {
			if strings.Contains(lowerContent, kw) {
				return MemoryTypeReference
			}
		}
		// If content is mostly URLs or short with URL, it's likely a reference
		words := strings.Fields(content)
		if len(words) < 10 {
			return MemoryTypeReference
		}
	}

	// Default to general
	return MemoryTypeGeneral
}

// ValidMemoryTypes returns all valid memory type values
func ValidMemoryTypes() []string {
	return []string{
		MemoryTypeGeneral,
		MemoryTypeDecision,
		MemoryTypeBugFix,
		MemoryTypePreference,
		MemoryTypePattern,
		MemoryTypeReference,
	}
}

// IsValidMemoryType checks if a type string is valid
func IsValidMemoryType(t string) bool {
	for _, valid := range ValidMemoryTypes() {
		if t == valid {
			return true
		}
	}
	return false
}
