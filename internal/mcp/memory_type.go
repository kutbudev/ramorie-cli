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
	MemoryTypeSkill      = "skill" // Procedural memory: trigger/steps/validation runbook
)

// weightedKeyword pairs a lowercase substring trigger with a confidence weight.
// Stronger, less-ambiguous phrases carry more weight so the scorer can resolve
// content that matches several buckets (e.g. "decided to always use yarn" hits
// BOTH a decision keyword and a preference keyword) by picking the bucket with
// the highest total signal rather than whichever list happened to be checked
// first.
type weightedKeyword struct {
	kw     string
	weight int
}

// memoryTypeKeywords maps each type to its weighted trigger phrases. Reference
// is handled separately because it is URL-gated (see DetectMemoryType).
//
// De-overlap notes:
//   - "best practice is" (preference, weight 2) deliberately outweighs the bare
//     "best practice" (pattern, weight 1) so an imperative rule phrased as a
//     best practice routes to preference, not pattern.
//   - imperative rule phrases ("always use", "never use", "should always") carry
//     the highest preference weight so durable user rules win over a co-occurring
//     decision verb ("decided to always use yarn" → preference).
var memoryTypeKeywords = map[string][]weightedKeyword{
	MemoryTypeDecision: {
		{"decision:", 3}, {"architectural decision", 3}, {"we chose", 3}, {"we decided", 3},
		{"adr", 2}, {"decided", 2}, {"chose", 2}, {"chosen", 2}, {"selected", 2},
		{"instead of", 2}, {"opted for", 2}, {"went with", 2}, {"we will use", 2},
		{"after considering", 2},
	},
	MemoryTypeBugFix: {
		{"root cause", 3}, {"the problem was", 3}, {"bug:", 3}, {"hotfix", 3},
		{"fix:", 2}, {"fixed", 2}, {"solved", 2}, {"resolved", 2}, {"error:", 2},
		{"issue:", 2}, {"patched", 2}, {"workaround", 2}, {"debugging", 2},
	},
	MemoryTypePreference: {
		{"always use", 3}, {"never use", 3}, {"should always", 3}, {"should never", 3},
		{"prefer", 2}, {"i like", 2}, {"we like", 2}, {"always", 2}, {"never", 2},
		{"convention:", 2}, {"standard:", 2}, {"rule:", 2}, {"best practice is", 2},
		{"recommended", 1},
	},
	MemoryTypePattern: {
		{"design pattern", 3}, {"architecture pattern", 3}, {"implementation pattern", 3},
		{"coding pattern", 3}, {"pattern:", 3}, {"template:", 2}, {"boilerplate", 2},
		{"structure:", 2}, {"best practice", 1},
	},
	MemoryTypeSkill: {
		{"runbook", 3}, {"skill:", 3}, {"step by step", 2}, {"step-by-step", 2},
		{"steps:", 2}, {"how to", 2}, {"how-to", 2}, {"recipe", 2}, {"procedure", 2},
	},
}

// referenceKeywords only count when the content also contains a URL — see
// DetectMemoryType. Keeping reference URL-gated avoids classifying prose that
// merely mentions "docs" or "guide" as a reference.
var referenceKeywords = []weightedKeyword{
	{"documentation", 2}, {"docs", 2}, {"reference", 2}, {"link:", 2}, {"see:", 2},
	{"url:", 2}, {"source:", 2}, {"article", 2}, {"tutorial", 2}, {"guide", 2},
}

// memoryTypePriority is the deterministic tie-break order applied when two
// buckets score equally. The first type in this slice with the maximum score
// wins (the scorer replaces only on a strictly-greater score).
var memoryTypePriority = []string{
	MemoryTypeDecision,
	MemoryTypeBugFix,
	MemoryTypePreference,
	MemoryTypeSkill,
	MemoryTypePattern,
	MemoryTypeReference,
}

var urlPattern = regexp.MustCompile(`https?://[^\s]+`)

// DetectMemoryType analyzes content and returns the most appropriate memory
// type by SCORING every bucket and selecting the strongest signal — not by
// returning the first list that matched. This fixes keyword shadowing where an
// early bucket (decision) would swallow content that more strongly belongs to a
// later bucket (preference), e.g. "decided to always use yarn".
func DetectMemoryType(content string) string {
	lowerContent := strings.ToLower(content)
	hasURL := urlPattern.MatchString(content)

	scores := make(map[string]int, len(memoryTypeKeywords)+1)
	for memType, keywords := range memoryTypeKeywords {
		for _, k := range keywords {
			if strings.Contains(lowerContent, k.kw) {
				scores[memType] += k.weight
			}
		}
	}

	// Reference is URL-gated: only score it when the content carries a link.
	if hasURL {
		for _, k := range referenceKeywords {
			if strings.Contains(lowerContent, k.kw) {
				scores[MemoryTypeReference] += k.weight
			}
		}
		// A short snippet that is mostly a URL is almost certainly a reference,
		// even without an explicit "docs"/"guide" keyword.
		if len(strings.Fields(content)) < 10 {
			scores[MemoryTypeReference] += 2
		}
	}

	best := MemoryTypeGeneral
	bestScore := 0
	for _, memType := range memoryTypePriority {
		if scores[memType] > bestScore {
			bestScore = scores[memType]
			best = memType
		}
	}
	if bestScore == 0 {
		return MemoryTypeGeneral
	}
	return best
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
		MemoryTypeSkill,
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
