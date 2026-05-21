package runtime

import (
	"context"
	"sort"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/config"
)

// DefaultModelProviderFilter returns the catalog provider id to use when listing models
// with no explicit provider (e.g. /config model picker after paste-key).
// Order: provider.json default → first configured deployment (stable sort by id).
func DefaultModelProviderFilter(ctx context.Context) string {
	rt, err := Load(ctx)
	if err != nil || rt == nil {
		return ""
	}
	if rt.Provider != nil {
		if p := config.DefaultProviderFromConfig(rt.Provider); p != "" {
			return catalog.CanonicalProviderID(p)
		}
	}
	rows, err := rt.DeploymentRows()
	if err != nil || len(rows) == 0 {
		return ""
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	for _, row := range rows {
		if row.Configured {
			if p := catalog.CanonicalProviderID(row.ProviderID); p != "" {
				return p
			}
		}
	}
	return ""
}
