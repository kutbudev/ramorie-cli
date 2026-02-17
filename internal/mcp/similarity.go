package mcp

import (
	"regexp"
	"sort"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/models"
)

// SimilarityThreshold is the minimum similarity score to consider memories similar
const SimilarityThreshold = 0.6

// AutoMergeThreshold is the similarity score above which memories are auto-merged
const AutoMergeThreshold = 0.85

// SimilarMemoryResult represents a memory that is similar to the input content
type SimilarMemoryResult struct {
	MemoryID    string  `json:"memory_id"`
	Content     string  `json:"content"`      // Truncated for display
	FullContent string  `json:"-"`            // Full content for merge operations
	Similarity  float64 `json:"similarity"`
}

// tokenize splits text into lowercase words, removing punctuation
func tokenize(text string) map[string]struct{} {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Remove punctuation and special characters, keep alphanumeric and spaces
	reg := regexp.MustCompile(`[^a-z0-9\s]`)
	text = reg.ReplaceAllString(text, " ")

	// Split by whitespace
	words := strings.Fields(text)

	// Create set of unique words
	wordSet := make(map[string]struct{})
	for _, word := range words {
		if len(word) > 1 { // Skip single-character words
			wordSet[word] = struct{}{}
		}
	}

	return wordSet
}

// JaccardSimilarity calculates the Jaccard similarity coefficient between two texts
// Returns a value between 0 (no overlap) and 1 (identical)
func JaccardSimilarity(a, b string) float64 {
	setA := tokenize(a)
	setB := tokenize(b)

	if len(setA) == 0 || len(setB) == 0 {
		return 0
	}

	// Calculate intersection
	intersection := 0
	for word := range setA {
		if _, exists := setB[word]; exists {
			intersection++
		}
	}

	// Calculate union (|A| + |B| - |A ∩ B|)
	union := len(setA) + len(setB) - intersection

	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// CheckSimilarMemories finds memories similar to the given content
// Returns a slice of similar memories sorted by similarity (highest first)
func CheckSimilarMemories(memories []models.Memory, content string, threshold float64) []SimilarMemoryResult {
	var similar []SimilarMemoryResult

	for _, memory := range memories {
		// Skip encrypted memories we can't read
		memoryContent := memory.Content
		if memory.IsEncrypted && memory.Content == "[Encrypted]" {
			continue
		}

		similarity := JaccardSimilarity(content, memoryContent)
		if similarity >= threshold {
			similar = append(similar, SimilarMemoryResult{
				MemoryID:    memory.ID.String(),
				Content:     truncateContent(memoryContent, 200),
				FullContent: memoryContent,
				Similarity:  similarity,
			})
		}
	}

	// Sort by similarity descending (highest first for auto-merge)
	sort.Slice(similar, func(i, j int) bool {
		return similar[i].Similarity > similar[j].Similarity
	})

	return similar
}

// mergeMemoryContent combines existing and new memory content intelligently
// If the new content is a subset of existing, keeps existing.
// Otherwise appends new unique information.
func mergeMemoryContent(existing, newContent string) string {
	// If new content is very similar, the existing is sufficient
	similarity := JaccardSimilarity(existing, newContent)
	if similarity >= 0.95 {
		// Almost identical, keep existing (it's already stored)
		return existing
	}

	// Find lines in new content that aren't in existing
	existingLines := strings.Split(strings.TrimSpace(existing), "\n")
	newLines := strings.Split(strings.TrimSpace(newContent), "\n")

	existingSet := make(map[string]bool)
	for _, line := range existingLines {
		existingSet[strings.TrimSpace(strings.ToLower(line))] = true
	}

	var additions []string
	for _, line := range newLines {
		normalized := strings.TrimSpace(strings.ToLower(line))
		if normalized == "" {
			continue
		}
		if !existingSet[normalized] {
			additions = append(additions, strings.TrimSpace(line))
		}
	}

	if len(additions) == 0 {
		return existing
	}

	// Append new unique lines
	return existing + "\n\n---\n" + strings.Join(additions, "\n")
}

// truncateContent truncates content to maxLen characters with ellipsis
func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen-3] + "..."
}

// ============================================================================
// Fuzzy Project Matching
// ============================================================================

// normalizeForMatch removes spaces, hyphens, underscores, dots for fuzzy matching.
// "Ramorie Frontend" -> "ramoriefrontend"
// "ramorie-frontend" -> "ramoriefrontend"
// "hangienstruman.com" -> "hangienstrumancom"
func normalizeForMatch(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, ".", "")
	return s
}

// projectMatch represents a fuzzy match result for project resolution
type projectMatch struct {
	Project    models.Project
	Confidence float64 // 0.0-1.0
	MatchType  string  // "normalized", "contains", "prefix"
}

// fuzzyMatchProjects finds projects that fuzzy-match the input string.
// Returns matches sorted by confidence DESC. Requires min 4 chars to prevent false positives.
func fuzzyMatchProjects(projects []models.Project, input string) []projectMatch {
	input = strings.TrimSpace(input)
	if len(input) < 4 {
		return nil
	}

	inputNorm := normalizeForMatch(input)
	if inputNorm == "" {
		return nil
	}

	var matches []projectMatch

	for _, p := range projects {
		projectNorm := normalizeForMatch(p.Name)
		if projectNorm == "" {
			continue
		}

		// 1. Normalized exact match (0.95)
		if inputNorm == projectNorm {
			matches = append(matches, projectMatch{
				Project:    p,
				Confidence: 0.95,
				MatchType:  "normalized",
			})
			continue
		}

		// 2. Substring contains (0.70-0.90)
		// Check if input is contained in project name or vice versa
		if strings.Contains(projectNorm, inputNorm) {
			coverage := float64(len(inputNorm)) / float64(len(projectNorm))
			score := 0.70 + coverage*0.20
			matches = append(matches, projectMatch{
				Project:    p,
				Confidence: score,
				MatchType:  "contains",
			})
			continue
		}
		if strings.Contains(inputNorm, projectNorm) {
			coverage := float64(len(projectNorm)) / float64(len(inputNorm))
			score := 0.70 + coverage*0.20
			matches = append(matches, projectMatch{
				Project:    p,
				Confidence: score,
				MatchType:  "contains",
			})
			continue
		}

		// 3. Prefix match (0.60-0.85)
		if strings.HasPrefix(projectNorm, inputNorm) || strings.HasPrefix(inputNorm, projectNorm) {
			shorter := len(inputNorm)
			longer := len(projectNorm)
			if shorter > longer {
				shorter, longer = longer, shorter
			}
			coverage := float64(shorter) / float64(longer)
			score := 0.60 + coverage*0.25
			matches = append(matches, projectMatch{
				Project:    p,
				Confidence: score,
				MatchType:  "prefix",
			})
			continue
		}
	}

	// Sort by confidence DESC
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Confidence > matches[j].Confidence
	})

	return matches
}
