package catalog

// MergeCatalogV1 merges models, offerings, providers, deployments, and aliases from src into dst.
// dst is modified in place and returned.
func MergeCatalogV1(dst, src *CatalogV1) *CatalogV1 {
	if dst == nil {
		return src
	}
	if src == nil {
		return dst
	}
	if dst.Providers == nil {
		dst.Providers = map[string]ProviderV1{}
	}
	for id, p := range src.Providers {
		if dst.Providers[id].ID == "" {
			dst.Providers[id] = p
		}
	}
	if dst.APIProtocols == nil {
		dst.APIProtocols = map[string]APIProtocolV1{}
	}
	for id, p := range src.APIProtocols {
		if dst.APIProtocols[id].ID == "" {
			dst.APIProtocols[id] = p
		}
	}
	if dst.Deployments == nil {
		dst.Deployments = map[string]DeploymentV1{}
	}
	for id, d := range src.Deployments {
		if dst.Deployments[id].ID == "" {
			dst.Deployments[id] = d
		}
	}
	if dst.Models == nil {
		dst.Models = map[string]ModelV1{}
	}
	for id, m := range src.Models {
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
