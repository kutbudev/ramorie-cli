// Package resolve adapter — bridges *api.Client to the resolve.* interfaces.
//
// Asymmetry note: *api.Client directly satisfies ProjectLister because both
// sides agree on the concrete []models.Project return type, so no adapter is
// needed for projects — callers can pass the *api.Client straight into
// ResolveProject. OrgLister, in contrast, returns []resolve.Organization
// (a structural view that intentionally avoids importing the api package),
// while *api.Client.ListOrganizations returns []api.Organization. The element
// type mismatch means the org side requires an explicit adapter, which is
// what OrgListerFromAPI provides below.
package resolve

import "github.com/kutbudev/ramorie-cli/internal/api"

// OrgListerFromAPI wraps an *api.Client so it satisfies OrgLister.
func OrgListerFromAPI(c *api.Client) OrgLister { return apiOrgAdapter{c} }

type apiOrgAdapter struct{ c *api.Client }

func (a apiOrgAdapter) ListOrganizations() ([]Organization, error) {
	apiOrgs, err := a.c.ListOrganizations()
	if err != nil {
		return nil, err
	}
	out := make([]Organization, 0, len(apiOrgs))
	for _, o := range apiOrgs {
		out = append(out, Organization{
			ID:   o.ID,
			Name: o.Name,
		})
	}
	return out, nil
}
