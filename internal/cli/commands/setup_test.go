package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kutbudev/ramorie-cli/internal/config"
)

// TestIsAuthConfigured_FalseWhenEmpty asserts that a missing or empty
// config file reports unauthenticated so runFullSetup will trigger the
// login step.
func TestIsAuthConfigured_FalseWhenEmpty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if isAuthConfigured() {
		t.Fatal("empty HOME should not be considered authenticated")
	}
}

// TestIsAuthConfigured_TrueAfterSave mirrors the positive case — a saved
// config with an API key flips the predicate.
func TestIsAuthConfigured_TrueAfterSave(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg := &config.Config{APIKey: "test-key-12345"}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Sanity-check the file we just wrote — ~/.ramorie/config.json.
	configPath := filepath.Join(tmp, ".ramorie", "config.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file should exist: %v", err)
	}

	if !isAuthConfigured() {
		t.Fatal("saved API key should make isAuthConfigured() true")
	}
}

// TestInstallHooksForAllDetected_DryRunIsSilent verifies dry-run mode never
// modifies the filesystem. We seed an isolated HOME so any accidental write
// would be visible.
func TestInstallHooksForAllDetected_DryRunIsSilent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if err := installHooksForAllDetected(true); err != nil {
		t.Fatalf("dry-run install should not error, got %v", err)
	}

	// The Claude Code installer would write to ~/.claude/settings.json — it
	// must not exist after a dry-run.
	if _, err := os.Stat(filepath.Join(tmp, ".claude", "settings.json")); err == nil {
		t.Error("dry-run must not create ~/.claude/settings.json")
	}
}

// TestInstallRulesForAllDetected_DryRunIsSilent twin of the hooks test.
func TestInstallRulesForAllDetected_DryRunIsSilent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if err := installRulesForAllDetected(true); err != nil {
		t.Fatalf("dry-run install should not error, got %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmp, ".codeium", "windsurf", "memories", "global_rules.md")); err == nil {
		t.Error("dry-run must not create windsurf rules file")
	}
}

// TestConfigShape_StableJSON is a guard against accidental field renames
// that would break SaveConfig/LoadConfig round-trips. We only spot-check
// the api_key key since that's the only field doctor/setup reads.
func TestConfigShape_StableJSON(t *testing.T) {
	cfg := &config.Config{APIKey: "abc"}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if raw["api_key"] != "abc" {
		t.Errorf("api_key field missing or renamed: %v", raw)
	}
}
