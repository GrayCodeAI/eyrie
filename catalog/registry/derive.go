package registry

import (
	"sort"
	"strings"
)

// CredentialSpec is paste-key / local setup metadata derived from ProviderSpec.
type CredentialSpec struct {
	ProviderID   string
	DisplayName  string
	DeploymentID string
	EnvVar       string
	ProbeKind    string
	ProbeBaseURL string
	RequiresKey  bool
	SortOrder    int
}

// SpecByProviderID finds a provider spec by id (accepts registry ids and catalog aliases like google→gemini).
func SpecByProviderID(id string) (ProviderSpec, bool) {
	return DefaultRegistry.Get(id)
}

func registryIDFromCatalogProvider(id string) string {
	switch strings.TrimSpace(id) {
	case "google":
		return "gemini"
	case "xai":
		return "grok"
	default:
		return id
	}
}

// SpecByEnvVar finds spec by primary credential env var.
func SpecByEnvVar(env string) (ProviderSpec, bool) {
	return DefaultRegistry.GetByEnv(env)
}

// DisplayName returns the UI label for a provider id.
func DisplayName(providerID string) string {
	if s, ok := SpecByProviderID(providerID); ok {
		return s.DisplayName
	}
	return providerID
}

// CredentialRegistry derives credential rows from provider specs.
func CredentialRegistry() []CredentialSpec {
	specs := DefaultRegistry.All()
	out := make([]CredentialSpec, len(specs))
	for i, s := range specs {
		out[i] = CredentialSpec{
			ProviderID:   s.ProviderID,
			DisplayName:  s.DisplayName,
			DeploymentID: s.DeploymentID,
			EnvVar:       s.CredentialEnv,
			ProbeKind:    string(s.ProbeKind),
			ProbeBaseURL: s.ProbeBaseURL,
			RequiresKey:  s.RequiresKey,
			SortOrder:    s.SortOrder,
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SortOrder < out[j].SortOrder })
	return out
}

// DeploymentEnvFallbacks derives seed env_fallbacks for registered deployments.
func DeploymentEnvFallbacks() map[string][]EnvFallback {
	return DefaultRegistry.DeploymentEnvFallbacks()
}

// LiveFetcherKeys returns live catalog provider keys that have fetchers.
func LiveFetcherKeys() []string {
	return DefaultRegistry.LiveFetcherKeys()
}

// LiveCatalogKeyForFetcher maps fetcher registry key to legacy catalog Providers map key.
func LiveCatalogKeyForFetcher(fetcherKey string) string {
	for _, s := range DefaultRegistry.All() {
		if s.LiveFetcherKey == fetcherKey {
			return s.LiveCatalogKey
		}
	}
	return fetcherKey
}

// CredentialPresent reports whether env satisfies this provider's discovery requirements.
func CredentialPresent(spec ProviderSpec, env map[string]string) bool {
	if spec.RequiresKey {
		return strings.TrimSpace(env[spec.CredentialEnv]) != ""
	}
	return strings.TrimSpace(env[spec.CredentialEnv]) != ""
}

// SpecForLiveFetcher returns the provider spec for a live fetcher key.
func SpecForLiveFetcher(fetcherKey string) (ProviderSpec, bool) {
	return DefaultRegistry.GetForLiveFetcher(fetcherKey)
}
