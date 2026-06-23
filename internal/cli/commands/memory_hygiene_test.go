package commands

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kutbudev/ramorie-cli/internal/models"
)

func TestAnalyzeMemoryHygiene_FindsRunbookAndSkillIssues(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	memories := []models.Memory{
		{
			ID:        uuid.New(),
			Content:   "Runbook: before:ios-build\nSteps: set OTHER_LDFLAGS then pod install.",
			Type:      "bug_fix",
			CreatedAt: now.AddDate(0, 0, -2),
			UpdatedAt: now.AddDate(0, 0, -2),
		},
		{
			ID:        uuid.New(),
			Content:   "iOS build procedure",
			Type:      "skill",
			CreatedAt: now.AddDate(0, 0, -1),
			UpdatedAt: now.AddDate(0, 0, -1),
		},
	}

	report := analyzeMemoryHygiene("proj", memories, now)

	if !hygieneHasKind(report, "runbook_prose") {
		t.Fatalf("expected runbook_prose issue, got %+v", report.Issues)
	}
	if !hygieneHasKind(report, "skill_unstructured") {
		t.Fatalf("expected skill_unstructured issue, got %+v", report.Issues)
	}
	if report.DryRun != true {
		t.Fatal("hygiene report must be dry-run")
	}
}

func TestAnalyzeMemoryHygiene_FindsStaleDuplicateAndThinLowValue(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	dup := "Exact duplicate memory with enough words to pass normalization safely"
	memories := []models.Memory{
		{
			ID:        uuid.New(),
			Content:   "tiny note",
			Type:      "general",
			CreatedAt: now.AddDate(0, 0, -20),
			UpdatedAt: now.AddDate(0, 0, -20),
		},
		{
			ID:        uuid.New(),
			Content:   dup,
			Type:      "general",
			CreatedAt: now.AddDate(0, 0, -45),
			UpdatedAt: now.AddDate(0, 0, -45),
		},
		{
			ID:        uuid.New(),
			Content:   dup,
			Type:      "general",
			CreatedAt: now.AddDate(0, 0, -44),
			UpdatedAt: now.AddDate(0, 0, -44),
		},
	}

	report := analyzeMemoryHygiene("proj", memories, now)

	for _, kind := range []string{"stale", "duplicate_exact", "thin_low_value"} {
		if !hygieneHasKind(report, kind) {
			t.Fatalf("expected %s issue, got %+v", kind, report.Issues)
		}
	}
}

func hygieneHasKind(report memoryHygieneReport, kind string) bool {
	for _, issue := range report.Issues {
		if issue.Kind == kind {
			return true
		}
	}
	return false
}
