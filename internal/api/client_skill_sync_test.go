package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// PR7 — Tests for UploadSkill / ListSkillsSyncState / PullSkillMarkdown.
//
// These mirror the existing client_skill_test.go style: spin a httptest
// server, route the relevant paths, assert on captured requests and
// decoded responses.
// ============================================================================

const testSkillMarkdown = `---
name: deploy-prod
description: Ship to production
---

# Deploy Prod
Steps to deploy.
`

func TestUploadSkill_PostsMarkdownAndDecodesResponse(t *testing.T) {
	var captured SkillUploadRequest
	var hits int

	skillID := uuid.New().String()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/skills/upload" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		hits++
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.Header().Set("Content-Type", "application/json")
		resp := SkillUploadResponse{
			ID:       skillID,
			Action:   "created",
			SyncHash: "abc123",
			SyncedAt: time.Now().UTC(),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	resp, err := c.UploadSkill(testSkillMarkdown, "/tmp/x.md", true)
	if err != nil {
		t.Fatalf("UploadSkill: %v", err)
	}
	if hits != 1 {
		t.Errorf("expected 1 hit on /skills/upload, got %d", hits)
	}
	if captured.Markdown != testSkillMarkdown {
		t.Errorf("markdown round-trip failed")
	}
	if captured.SourcePath != "/tmp/x.md" {
		t.Errorf("source_path round-trip failed: %q", captured.SourcePath)
	}
	if !captured.Overwrite {
		t.Errorf("overwrite flag should round-trip")
	}
	if resp.ID != skillID {
		t.Errorf("response id mismatch: %q vs %q", resp.ID, skillID)
	}
	if resp.Action != "created" {
		t.Errorf("response action mismatch: %q", resp.Action)
	}
}

func TestUploadSkill_RejectsEmptyMarkdown(t *testing.T) {
	c := newTestClient(httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be called for empty markdown")
	})))
	if _, err := c.UploadSkill("   ", "", false); err == nil {
		t.Fatal("expected error on empty markdown")
	}
}

func TestListSkillsSyncState_ParsesEnvelope(t *testing.T) {
	id := uuid.New().String()
	now := time.Now().UTC()
	hash := "deadbeef"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/skills/sync-state" {
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := SkillSyncStateResponse{
			Count: 1,
			Items: []SkillSyncStateItem{{
				ID:           id,
				Name:         "deploy-prod",
				SyncHash:     &hash,
				SyncedAt:     &now,
				LastModified: now,
			}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.ListSkillsSyncState()
	if err != nil {
		t.Fatalf("ListSkillsSyncState: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "deploy-prod" {
		t.Errorf("name mismatch: %q", items[0].Name)
	}
	if items[0].SyncHash == nil || *items[0].SyncHash != hash {
		t.Errorf("sync_hash round-trip failed")
	}
}

func TestPullSkillMarkdown_ReturnsRawBody(t *testing.T) {
	skillID := uuid.New().String()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/memories/"+skillID+"/raw-markdown" {
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		_, _ = io.WriteString(w, testSkillMarkdown)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	body, err := c.PullSkillMarkdown(skillID)
	if err != nil {
		t.Fatalf("PullSkillMarkdown: %v", err)
	}
	if !strings.Contains(body, "name: deploy-prod") {
		t.Errorf("body missing frontmatter; got %q", body)
	}
}

func TestPullSkillMarkdown_RejectsEmptyID(t *testing.T) {
	c := newTestClient(httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be called for empty id")
	})))
	if _, err := c.PullSkillMarkdown("  "); err == nil {
		t.Fatal("expected error on empty id")
	}
}
