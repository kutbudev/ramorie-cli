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

func TestNormalizeForMatch(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Ramorie Frontend", "ramoriefrontend"},
		{"ramorie-frontend", "ramoriefrontend"},
		{"hangienstruman.com", "hangienstrumancom"},
		{"orkai.io", "orkaiio"},
		{"my_project", "myproject"},
		{"UPPER CASE", "uppercase"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeForMatch(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeForMatch(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFuzzyMatchProjects(t *testing.T) {
	projects := []models.Project{
		{ID: uuid.New(), Name: "hangienstruman.com"},
		{ID: uuid.New(), Name: "ramorie-frontend"},
		{ID: uuid.New(), Name: "ramorie-backend"},
		{ID: uuid.New(), Name: "orkai.io"},
	}

	t.Run("substring match: hangienstruman -> hangienstruman.com", func(t *testing.T) {
		matches := fuzzyMatchProjects(projects, "hangienstruman")
		if len(matches) == 0 {
			t.Fatal("Expected at least one match")
		}
		if matches[0].Project.Name != "hangienstruman.com" {
			t.Errorf("Expected top match 'hangienstruman.com', got %q", matches[0].Project.Name)
		}
		if matches[0].Confidence < 0.70 {
			t.Errorf("Expected confidence >= 0.70, got %f", matches[0].Confidence)
		}
		if matches[0].MatchType != "contains" {
			t.Errorf("Expected match type 'contains', got %q", matches[0].MatchType)
		}
	})

	t.Run("normalized exact match: ramorie frontend -> ramorie-frontend", func(t *testing.T) {
		matches := fuzzyMatchProjects(projects, "ramorie frontend")
		if len(matches) == 0 {
			t.Fatal("Expected at least one match")
		}
		if matches[0].Project.Name != "ramorie-frontend" {
			t.Errorf("Expected top match 'ramorie-frontend', got %q", matches[0].Project.Name)
		}
		if matches[0].Confidence != 0.95 {
			t.Errorf("Expected confidence 0.95, got %f", matches[0].Confidence)
		}
		if matches[0].MatchType != "normalized" {
			t.Errorf("Expected match type 'normalized', got %q", matches[0].MatchType)
		}
	})

	t.Run("exact case insensitive already handled by caller", func(t *testing.T) {
		// fuzzyMatchProjects should still find normalized exact
		matches := fuzzyMatchProjects(projects, "Ramorie-Frontend")
		if len(matches) == 0 {
			t.Fatal("Expected at least one match")
		}
		if matches[0].Project.Name != "ramorie-frontend" {
			t.Errorf("Expected 'ramorie-frontend', got %q", matches[0].Project.Name)
		}
	})

	t.Run("prefix match: ramorie -> ramorie-frontend and ramorie-backend", func(t *testing.T) {
		matches := fuzzyMatchProjects(projects, "ramorie")
		if len(matches) < 2 {
			t.Fatalf("Expected at least 2 matches, got %d", len(matches))
		}
		// Both ramorie-frontend and ramorie-backend should match
		names := make(map[string]bool)
		for _, m := range matches {
			names[m.Project.Name] = true
		}
		if !names["ramorie-frontend"] {
			t.Error("Expected ramorie-frontend in matches")
		}
		if !names["ramorie-backend"] {
			t.Error("Expected ramorie-backend in matches")
		}
	})

	t.Run("no match for unrelated input", func(t *testing.T) {
		matches := fuzzyMatchProjects(projects, "nonexistent")
		if len(matches) != 0 {
			t.Errorf("Expected no matches, got %d", len(matches))
		}
	})

	t.Run("input too short is rejected", func(t *testing.T) {
		matches := fuzzyMatchProjects(projects, "abc")
		if len(matches) != 0 {
			t.Errorf("Expected no matches for short input, got %d", len(matches))
		}
	})

	t.Run("empty input returns nil", func(t *testing.T) {
		matches := fuzzyMatchProjects(projects, "")
		if matches != nil {
			t.Errorf("Expected nil for empty input, got %v", matches)
		}
	})

	t.Run("sorted by confidence descending", func(t *testing.T) {
		matches := fuzzyMatchProjects(projects, "ramorie")
		for i := 1; i < len(matches); i++ {
			if matches[i].Confidence > matches[i-1].Confidence {
				t.Errorf("Matches not sorted by confidence: %f > %f at index %d",
					matches[i].Confidence, matches[i-1].Confidence, i)
			}
		}
	})
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
