package resolve

import "github.com/kutbudev/ramorie-cli/internal/api"

// FromAPI wraps an *api.Client so it satisfies OrgLister.
func FromAPI(c *api.Client) OrgLister { return apiOrgAdapter{c} }

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
