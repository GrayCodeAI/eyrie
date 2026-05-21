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
	KeyPrefixes  []string
	ProbeKind    string
	ProbeBaseURL string
	RequiresKey  bool
	SortOrder    int
}

// SpecByProviderID finds a provider spec by id (accepts registry ids and catalog aliases like google→gemini).
func SpecByProviderID(id string) (ProviderSpec, bool) {
	id = strings.TrimSpace(id)
	for _, s := range All() {
		if s.ProviderID == id {
			return s, true
		}
	}
	if alt := registryIDFromCatalogProvider(id); alt != id {
		for _, s := range All() {
			if s.ProviderID == alt {
				return s, true
			}
		}
	}
	return ProviderSpec{}, false
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
	env = strings.TrimSpace(env)
	for _, s := range All() {
		if s.CredentialEnv == env {
			return s, true
		}
	}
	return ProviderSpec{}, false
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
	specs := All()
	out := make([]CredentialSpec, len(specs))
	for i, s := range specs {
		out[i] = CredentialSpec{
			ProviderID:   s.ProviderID,
			DisplayName:  s.DisplayName,
			DeploymentID: s.DeploymentID,
			EnvVar:       s.CredentialEnv,
			KeyPrefixes:  append([]string(nil), s.KeyPrefixes...),
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
	out := map[string][]EnvFallback{}
	for _, s := range All() {
		var fb []EnvFallback
		if s.RequiresKey && s.CredentialEnv != "" {
			fb = append(fb, EnvFallback{Field: "api_key", Env: []string{s.CredentialEnv}})
		}
		if !s.RequiresKey && s.CredentialEnv != "" {
			fb = append(fb, EnvFallback{Field: "base_url", Env: []string{s.CredentialEnv}})
		}
		if len(s.BaseURLEnv) > 0 {
			found := false
			for _, f := range fb {
				if f.Field == "base_url" {
					found = true
					break
				}
			}
			if !found {
				fb = append(fb, EnvFallback{Field: "base_url", Env: append([]string(nil), s.BaseURLEnv...)})
			} else if len(s.BaseURLEnv) > 0 {
				// merge base url envs
				for i, f := range fb {
					if f.Field == "base_url" {
						seen := map[string]bool{}
						for _, e := range f.Env {
							seen[e] = true
						}
						for _, e := range s.BaseURLEnv {
							if !seen[e] {
								fb[i].Env = append(fb[i].Env, e)
							}
						}
					}
				}
			}
		}
		if s.ProviderID == "ollama" {
			fb = append(fb, EnvFallback{Field: "api_key", Env: []string{"OLLAMA_API_KEY", "OPENAI_API_KEY"}})
		}
		out[s.DeploymentID] = fb
	}
	return out
}

// LiveFetcherKeys returns live catalog provider keys that have fetchers.
func LiveFetcherKeys() []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range All() {
		if s.LiveFetcherKey == "" || seen[s.LiveFetcherKey] {
			continue
		}
		seen[s.LiveFetcherKey] = true
		out = append(out, s.LiveFetcherKey)
	}
	sort.Strings(out)
	return out
}

// LiveCatalogKeyForFetcher maps fetcher registry key to legacy catalog Providers map key.
func LiveCatalogKeyForFetcher(fetcherKey string) string {
	for _, s := range All() {
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
	for _, s := range All() {
		if s.LiveFetcherKey == fetcherKey {
			return s, true
		}
	}
	return ProviderSpec{}, false
}
