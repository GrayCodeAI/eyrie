package discover

import (
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
)

// MergePolicy controls catalog merge behavior when enriching from live APIs.
type MergePolicy struct {
	PreferLive                 bool
	ReplaceDeploymentOfferings []string
}

// MergeCatalogV1 merges models, offerings, providers, deployments, and aliases from src into dst.
func MergeCatalogV1(dst, src *catalog.CatalogV1) *catalog.CatalogV1 {
	return MergeCatalogV1WithPolicy(dst, src, MergePolicy{})
}

// MergeCatalogV1WithPolicy merges with optional live-preference for existing model IDs.
func MergeCatalogV1WithPolicy(dst, src *catalog.CatalogV1, policy MergePolicy) *catalog.CatalogV1 {
	if dst == nil {
		return src
	}
	if src == nil {
		return dst
	}
	if dst.Providers == nil {
		dst.Providers = map[string]catalog.ProviderV1{}
	}
	for id, p := range src.Providers {
		if dst.Providers[id].ID == "" {
			dst.Providers[id] = p
		}
	}
	if dst.APIProtocols == nil {
		dst.APIProtocols = map[string]catalog.APIProtocolV1{}
	}
	for id, p := range src.APIProtocols {
		if dst.APIProtocols[id].ID == "" {
			dst.APIProtocols[id] = p
		}
	}
	if dst.Deployments == nil {
		dst.Deployments = map[string]catalog.DeploymentV1{}
	}
	for id, d := range src.Deployments {
		if dst.Deployments[id].ID == "" {
			dst.Deployments[id] = d
		}
	}
	if dst.Models == nil {
		dst.Models = map[string]catalog.ModelV1{}
	}
	if len(policy.ReplaceDeploymentOfferings) > 0 {
		remove := map[string]bool{}
		for _, dep := range policy.ReplaceDeploymentOfferings {
			if dep = strings.TrimSpace(dep); dep != "" {
				remove[dep] = true
			}
		}
		if len(remove) > 0 {
			filtered := dst.Offerings[:0]
			for _, o := range dst.Offerings {
				if !remove[o.DeploymentID] {
					filtered = append(filtered, o)
				}
			}
			dst.Offerings = filtered
		}
	}
	for id, m := range src.Models {
		if existing, ok := dst.Models[id]; ok && policy.PreferLive {
			if m.ContextWindow > 0 {
				existing.ContextWindow = m.ContextWindow
			}
			if m.MaxOutput > 0 {
				existing.MaxOutput = m.MaxOutput
			}
			if strings.TrimSpace(m.Name) != "" {
				existing.Name = m.Name
			}
			dst.Models[id] = existing
			continue
		}
		if dst.Models[id].ID == "" {
			dst.Models[id] = m
		}
	}
	seen := map[string]bool{}
	for _, o := range dst.Offerings {
		seen[o.ID] = true
	}
	for _, o := range src.Offerings {
		if o.ID == "" || seen[o.ID] {
			continue
		}
		seen[o.ID] = true
		dst.Offerings = append(dst.Offerings, o)
	}
	if dst.Aliases == nil {
		dst.Aliases = map[string]string{}
	}
	for alias, canonical := range src.Aliases {
		if dst.Aliases[alias] == "" {
			dst.Aliases[alias] = canonical
		}
	}
	return dst
}
