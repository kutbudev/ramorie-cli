// Package rules installs the Ramorie Persistent Memory Protocol into AI
// editors that consume rule files instead of shell hooks (Cursor, Windsurf).
// The installers all share a managed-block convention so re-installing
// replaces the prior block instead of duplicating it.
package rules

import (
	"fmt"
	"regexp"
	"strings"
)

// Marker sentinels delimit the Ramorie-managed region inside otherwise
// hand-edited rule files. Bumping `v=` lets readers spot stale blocks.
const (
	MarkerStart = "<!-- ramorie:managed:start v=1 -->"
	MarkerEnd   = "<!-- ramorie:managed:end -->"
)

// Installer is the contract every editor integration implements.
type Installer interface {
	// Name is a stable identifier ("cursor", "windsurf").
	Name() string
	// Detect reports whether the editor's config root is present.
	Detect() bool
	// RulesPath is the file we mutate.
	RulesPath() string
	// Install writes `text` inside the managed block, replacing any prior
	// managed block. Idempotent — three calls yield identical content.
	Install(text string) error
	// Uninstall removes the managed block (leaves surrounding text intact).
	Uninstall() error
	// Status reports whether a managed block is currently installed and,
	// if so, its version marker.
	Status() (installed bool, version string, err error)
}

// managedBlockRegexp matches the start marker (with optional version),
// everything in between, and the end marker. (?s) so `.` spans newlines.
var managedBlockRegexp = regexp.MustCompile(
	`(?s)<!-- ramorie:managed:start[^>]*-->.*?<!-- ramorie:managed:end -->`,
)

// versionRegexp pulls the v=N token out of a start marker for Status().
var versionRegexp = regexp.MustCompile(`v=([0-9]+)`)

// wrapManaged builds the marker-delimited block. Leading/trailing newlines
// make the output stay readable when injected into a markdown rules file.
func wrapManaged(text string) string {
	return fmt.Sprintf("%s\n%s\n%s", MarkerStart, strings.TrimSpace(text), MarkerEnd)
}

// upsertManagedBlock replaces an existing block or appends a new one.
// Returns the new file content. Used by every text-file installer.
func upsertManagedBlock(existing, text string) string {
	block := wrapManaged(text)
	if managedBlockRegexp.MatchString(existing) {
		return managedBlockRegexp.ReplaceAllString(existing, block)
	}
	// Append with a separating blank line if the file already has content.
	trimmed := strings.TrimRight(existing, "\n")
	if trimmed == "" {
		return block + "\n"
	}
	return trimmed + "\n\n" + block + "\n"
}

// removeManagedBlock strips any managed region (and the surrounding blank
// line we inserted on append).
func removeManagedBlock(existing string) string {
	if !managedBlockRegexp.MatchString(existing) {
		return existing
	}
	out := managedBlockRegexp.ReplaceAllString(existing, "")
	// Collapse the double blank line left behind by removing an appended
	// block so the file stays tidy.
	out = regexp.MustCompile(`\n{3,}`).ReplaceAllString(out, "\n\n")
	return strings.TrimRight(out, "\n") + "\n"
}

// extractVersion returns the v=N value of the first managed block in
// `existing`, or "" if no block is present.
func extractVersion(existing string) string {
	loc := managedBlockRegexp.FindStringIndex(existing)
	if loc == nil {
		return ""
	}
	header := existing[loc[0]:]
	if newline := strings.IndexByte(header, '\n'); newline > 0 {
		header = header[:newline]
	}
	m := versionRegexp.FindStringSubmatch(header)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}
