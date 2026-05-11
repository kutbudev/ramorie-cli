package commands

import (
	"reflect"
	"testing"
)

func TestParseFindArgs_AllowsFlagsAfterSearchTerm(t *testing.T) {
	got, err := parseFindArgs([]string{
		"Ramorie", "memory", "health",
		"--project", "workflow",
		"--limit", "10",
		"--budget=5000",
		"--hyde", "off",
		"--rerank=off",
		"--intent", "generic",
		"--entity-hops", "2",
		"--include-superseded",
		"--fast=false",
		"--types", "decision",
		"-t", "bug_fix",
		"--tags", "mcp",
	})
	if err != nil {
		t.Fatalf("parseFindArgs: %v", err)
	}
	if want := []string{"Ramorie", "memory", "health"}; !reflect.DeepEqual(got.TermParts, want) {
		t.Fatalf("term parts: got %v want %v", got.TermParts, want)
	}
	if got.Project == nil || *got.Project != "workflow" {
		t.Fatalf("project: got %v", got.Project)
	}
	if got.Limit == nil || *got.Limit != 10 {
		t.Fatalf("limit: got %v", got.Limit)
	}
	if got.Budget == nil || *got.Budget != 5000 {
		t.Fatalf("budget: got %v", got.Budget)
	}
	if got.HyDE == nil || *got.HyDE != "off" {
		t.Fatalf("hyde: got %v", got.HyDE)
	}
	if got.Rerank == nil || *got.Rerank != "off" {
		t.Fatalf("rerank: got %v", got.Rerank)
	}
	if got.Intent == nil || *got.Intent != "generic" {
		t.Fatalf("intent: got %v", got.Intent)
	}
	if got.EntityHops == nil || *got.EntityHops != 2 {
		t.Fatalf("entity hops: got %v", got.EntityHops)
	}
	if got.IncludeSuperseded == nil || *got.IncludeSuperseded != true {
		t.Fatalf("include superseded: got %v", got.IncludeSuperseded)
	}
	if got.FastMode == nil || *got.FastMode != false {
		t.Fatalf("fast mode: got %v", got.FastMode)
	}
	if want := []string{"decision", "bug_fix"}; !reflect.DeepEqual(got.Types, want) {
		t.Fatalf("types: got %v want %v", got.Types, want)
	}
	if want := []string{"mcp"}; !reflect.DeepEqual(got.Tags, want) {
		t.Fatalf("tags: got %v want %v", got.Tags, want)
	}
}

func TestParseFindArgs_DoubleDashKeepsFlagLikeSearchText(t *testing.T) {
	got, err := parseFindArgs([]string{"why", "--", "--project", "is", "literal"})
	if err != nil {
		t.Fatalf("parseFindArgs: %v", err)
	}
	want := []string{"why", "--project", "is", "literal"}
	if !reflect.DeepEqual(got.TermParts, want) {
		t.Fatalf("term parts: got %v want %v", got.TermParts, want)
	}
}

func TestParseFindArgs_RejectsUnknownTrailingFlag(t *testing.T) {
	if _, err := parseFindArgs([]string{"query", "--unknown"}); err == nil {
		t.Fatal("expected unknown trailing flag error")
	}
}
