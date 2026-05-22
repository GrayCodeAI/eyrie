package credential

import (
	"context"
	"sort"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

// CredentialInference is one provider/deployment match for a pasted API key (no secret).
type CredentialInference struct {
	ProviderID   string `json:"provider_id"`
	DeploymentID string `json:"deployment_id"`
	EnvVar       string `json:"env_var"`
	DisplayName  string `json:"display_name"`
}

// InferCredentialsFromAPIKey returns prefix-inferred candidates (legacy API; prefer ResolveCredential).
func InferCredentialsFromAPIKey(ctx context.Context, secret string) []CredentialInference {
	res := ResolveCredential(ctx, secret)
	if !res.FormatOK {
		return nil
	}
	var out []CredentialInference
	for _, opt := range res.Providers {
		if !opt.Inferred {
			continue
		}
		out = append(out, InferenceFromOption(opt))
	}
	if len(out) > 0 {
		return out
	}
	// Fallback: catalog-backed inference for providers not in registry prefixes.
	return inferFromCatalog(ctx, secret)
}

func inferFromCatalog(ctx context.Context, secret string) []CredentialInference {
	if err := ValidateKeyFormat(secret); err != nil {
		return nil
	}
	compiled, err := catalog.LoadCatalogForDiscovery(ctx)
	if err != nil || compiled == nil {
		return nil
	}
	providerIDs := matchedProviderIDsFromRegistry(secret)
	if len(providerIDs) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var out []CredentialInference
	for _, pid := range providerIDs {
		for _, depID := range deploymentIDsForProvider(compiled, pid) {
			env := catalog.PrimaryAPIKeyEnvForDeployment(compiled, depID)
			if env == "" || seen[env] || !isProviderAPIKeyEnv(env) {
				continue
			}
			if err := ValidateCredentialSecret(env, secret); err != nil {
				continue
			}
			seen[env] = true
			out = append(out, CredentialInference{
				ProviderID:   pid,
				DeploymentID: depID,
				EnvVar:       env,
				DisplayName:  inferenceDisplayName(compiled, depID, pid),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ProviderID != out[j].ProviderID {
			return out[i].ProviderID < out[j].ProviderID
		}
		return out[i].DeploymentID < out[j].DeploymentID
	})
	return out
}

func deploymentIDsForProvider(compiled *catalog.CompiledCatalogV1, providerID string) []string {
	if compiled == nil || compiled.Catalog == nil {
		return []string{providerID + "-direct"}
	}
	providerID = catalog.CanonicalProviderID(providerID)
	preferred := []string{providerID + "-direct", providerID}
	var out []string
	seen := map[string]bool{}
	add := func(id string) {
		if id == "" || seen[id] {
			return
		}
		if _, ok := compiled.Catalog.Deployments[id]; !ok {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	for _, id := range preferred {
		add(id)
	}
	for id, dep := range compiled.Catalog.Deployments {
		if catalog.CanonicalProviderID(dep.ProviderID) == providerID {
			add(id)
		}
	}
	return out
}

func isProviderAPIKeyEnv(env string) bool {
	return strings.Contains(strings.ToUpper(strings.TrimSpace(env)), "API_KEY") ||
		strings.Contains(strings.ToUpper(strings.TrimSpace(env)), "TOKEN")
}

func inferenceDisplayName(compiled *catalog.CompiledCatalogV1, deploymentID, providerID string) string {
	if spec, ok := registry.DefaultRegistry.Get(providerID); ok {
		return spec.DisplayName
	}
	if compiled != nil && compiled.Catalog != nil {
		if dep, ok := compiled.Catalog.Deployments[deploymentID]; ok {
			if name := strings.TrimSpace(dep.Name); name != "" {
				return name
			}
		}
	}
	return providerID
}
