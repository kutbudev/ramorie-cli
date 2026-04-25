// Package resolve maps user-facing identifiers (name, short UUID, full UUID)
// to canonical full UUID strings. All CLI commands MUST use these helpers
// instead of duplicating inline list-and-match logic.
package resolve

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kutbudev/ramorie-cli/internal/models"
)

// ProjectLister is the minimal API surface ResolveProject needs. Implemented
// by *api.Client; injectable in tests.
type ProjectLister interface {
	ListProjects(orgID ...string) ([]models.Project, error)
}

// ResolveProject returns the full UUID string for a project identified by
// arg, which may be: full UUID, short prefix (>=4 chars), exact name
// (case-insensitive), or unique substring of the name.
func ResolveProject(arg string, l ProjectLister) (string, error) {
	if arg == "" {
		return "", errors.New("project identifier is required")
	}
	projects, err := l.ListProjects()
	if err != nil {
		return "", fmt.Errorf("could not fetch projects: %w", err)
	}

	argLower := strings.ToLower(arg)

	for _, p := range projects {
		if p.ID.String() == arg {
			return p.ID.String(), nil
		}
	}
	for _, p := range projects {
		if strings.EqualFold(p.Name, arg) {
			return p.ID.String(), nil
		}
	}
	if len(arg) >= 4 {
		var prefixHits []models.Project
		for _, p := range projects {
			if strings.HasPrefix(p.ID.String(), arg) {
				prefixHits = append(prefixHits, p)
			}
		}
		if len(prefixHits) == 1 {
			return prefixHits[0].ID.String(), nil
		}
		if len(prefixHits) > 1 {
			return "", ambiguousProjectErr(arg, prefixHits)
		}
	}
	var nameHits []models.Project
	for _, p := range projects {
		if strings.Contains(strings.ToLower(p.Name), argLower) {
			nameHits = append(nameHits, p)
		}
	}
	if len(nameHits) == 1 {
		return nameHits[0].ID.String(), nil
	}
	if len(nameHits) > 1 {
		return "", ambiguousProjectErr(arg, nameHits)
	}

	return "", fmt.Errorf("project %q not found — run `ramorie project list` to see available", arg)
}

func ambiguousProjectErr(arg string, hits []models.Project) error {
	names := make([]string, 0, len(hits))
	for _, h := range hits {
		names = append(names, fmt.Sprintf("%s (%s)", h.Name, h.ID.String()[:8]))
	}
	return fmt.Errorf("project %q is ambiguous — matches: %s", arg, strings.Join(names, ", "))
}

// OrgLister is the minimal API for ResolveOrg.
type OrgLister interface {
	ListOrganizations() ([]Organization, error)
}

// Organization is the structural view ResolveOrg needs (avoids importing api package).
// Callers adapt with FromAPI in api_adapter.go.
type Organization struct {
	ID   string
	Name string
}

// ResolveOrg mirrors ResolveProject for organizations.
func ResolveOrg(arg string, l OrgLister) (string, error) {
	if arg == "" {
		return "", errors.New("organization identifier is required")
	}
	orgs, err := l.ListOrganizations()
	if err != nil {
		return "", fmt.Errorf("could not fetch organizations: %w", err)
	}

	argLower := strings.ToLower(arg)

	for _, o := range orgs {
		if o.ID == arg {
			return o.ID, nil
		}
	}
	for _, o := range orgs {
		if strings.EqualFold(o.Name, arg) {
			return o.ID, nil
		}
	}
	if len(arg) >= 4 {
		var hits []Organization
		for _, o := range orgs {
			if strings.HasPrefix(o.ID, arg) {
				hits = append(hits, o)
			}
		}
		if len(hits) == 1 {
			return hits[0].ID, nil
		}
		if len(hits) > 1 {
			return "", ambiguousOrgErr(arg, hits)
		}
	}
	var nameHits []Organization
	for _, o := range orgs {
		if strings.Contains(strings.ToLower(o.Name), argLower) {
			nameHits = append(nameHits, o)
		}
	}
	if len(nameHits) == 1 {
		return nameHits[0].ID, nil
	}
	if len(nameHits) > 1 {
		return "", ambiguousOrgErr(arg, nameHits)
	}
	return "", fmt.Errorf("organization %q not found — run `ramorie org list` to see available", arg)
}

func ambiguousOrgErr(arg string, hits []Organization) error {
	names := make([]string, 0, len(hits))
	for _, h := range hits {
		shortID := h.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		names = append(names, fmt.Sprintf("%s (%s)", h.Name, shortID))
	}
	return fmt.Errorf("organization %q is ambiguous — matches: %s", arg, strings.Join(names, ", "))
}

// ErrNotFullUUID is returned by ResolveID when arg is not a 36-char dashed UUID.
var ErrNotFullUUID = errors.New("not a full UUID")

// ResolveID returns arg unchanged if it looks like a full UUID. Otherwise
// returns ErrNotFullUUID — callers handle short IDs via per-resource lookups.
func ResolveID(arg string) (string, error) {
	if arg == "" {
		return "", errors.New("identifier is required")
	}
	if len(arg) == 36 && strings.Count(arg, "-") == 4 {
		return arg, nil
	}
	return "", ErrNotFullUUID
}
