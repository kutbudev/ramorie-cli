package mcp

import (
	"regexp"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/models"
)

// SimilarityThreshold is the minimum similarity score to consider memories similar
const SimilarityThreshold = 0.6

// SimilarMemoryResult represents a memory that is similar to the input content
type SimilarMemoryResult struct {
	MemoryID   string  `json:"memory_id"`
	Content    string  `json:"content"`
	Similarity float64 `json:"similarity"`
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
// Returns a slice of similar memories that exceed the threshold
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
				MemoryID:   memory.ID.String(),
				Content:    truncateContent(memoryContent, 200),
				Similarity: similarity,
			})
		}
	}

	return similar
}

// truncateContent truncates content to maxLen characters with ellipsis
func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen-3] + "..."
}
