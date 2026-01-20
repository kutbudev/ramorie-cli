package mcp

import (
	"testing"

	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

func TestJaccardSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		minScore float64
		maxScore float64
	}{
		{
			name:     "identical strings",
			a:        "Use Redis for caching",
			b:        "Use Redis for caching",
			minScore: 0.99,
			maxScore: 1.0,
		},
		{
			name:     "very similar strings",
			a:        "Use Redis for caching",
			b:        "Redis is great for caching",
			minScore: 0.5, // Should be above threshold
			maxScore: 0.8,
		},
		{
			name:     "somewhat similar strings",
			a:        "Use Redis for caching in production",
			b:        "Redis caching is useful",
			minScore: 0.2,
			maxScore: 0.5,
		},
		{
			name:     "completely different strings",
			a:        "The quick brown fox",
			b:        "PostgreSQL database migration",
			minScore: 0.0,
			maxScore: 0.1,
		},
		{
			name:     "empty string handling",
			a:        "",
			b:        "Some content",
			minScore: 0.0,
			maxScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := JaccardSimilarity(tt.a, tt.b)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("JaccardSimilarity(%q, %q) = %v, want between %v and %v",
					tt.a, tt.b, score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestCheckSimilarMemories(t *testing.T) {
	memories := []models.Memory{
		{
			ID:      uuid.New(),
			Content: "Use Redis for caching data",
		},
		{
			ID:      uuid.New(),
			Content: "PostgreSQL is a relational database",
		},
		{
			ID:      uuid.New(),
			Content: "API authentication uses JWT tokens",
		},
	}

	// Test finding similar memory
	similar := CheckSimilarMemories(memories, "Redis is great for caching", 0.4)
	if len(similar) == 0 {
		t.Error("Expected to find similar memories for Redis caching content")
	}

	// Test no similar memories found
	similar = CheckSimilarMemories(memories, "GraphQL schema design patterns", 0.6)
	if len(similar) != 0 {
		t.Errorf("Expected no similar memories, got %d", len(similar))
	}

	// Test high threshold filters out borderline matches
	similar = CheckSimilarMemories(memories, "Redis is great for caching", 0.9)
	if len(similar) != 0 {
		t.Errorf("Expected no matches at high threshold, got %d", len(similar))
	}
}

func TestTokenize(t *testing.T) {
	result := tokenize("Hello, World! This is a TEST.")

	// Should contain lowercase words
	if _, exists := result["hello"]; !exists {
		t.Error("Expected 'hello' in tokenized result")
	}
	if _, exists := result["world"]; !exists {
		t.Error("Expected 'world' in tokenized result")
	}
	if _, exists := result["test"]; !exists {
		t.Error("Expected 'test' in tokenized result")
	}

	// Should not contain single-character words
	if _, exists := result["a"]; exists {
		t.Error("Should not contain single-character word 'a'")
	}
}

func TestTruncateContent(t *testing.T) {
	// Short content should not be truncated
	short := "Short text"
	if truncateContent(short, 50) != short {
		t.Error("Short content should not be truncated")
	}

	// Long content should be truncated with ellipsis
	long := "This is a very long piece of content that should be truncated"
	truncated := truncateContent(long, 20)
	if len(truncated) != 20 {
		t.Errorf("Expected length 20, got %d", len(truncated))
	}
	if truncated[len(truncated)-3:] != "..." {
		t.Error("Expected ellipsis at end of truncated content")
	}
}
