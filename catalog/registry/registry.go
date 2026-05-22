package registry

import (
	"sort"
	"strings"
	"sync"
)

// ProviderRegistry is the single source of truth for all provider specs.
type ProviderRegistry struct {
	mu    sync.RWMutex
	specs []ProviderSpec
	byID  map[string]*ProviderSpec
}

// DefaultRegistry is the global default provider registry, populated at init.
var DefaultRegistry = NewProviderRegistry()

// NewProviderRegistry creates a new empty registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		byID: make(map[string]*ProviderSpec),
	}
}

// Register adds a spec to the registry. If a spec with the same ProviderID
// already exists, it is replaced.
func (r *ProviderRegistry) Register(spec ProviderSpec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.byID[spec.ProviderID]; ok {
		*existing = spec
		return
	}
	r.byID[spec.ProviderID] = &spec
	r.specs = append(r.specs, spec)
}

// Get returns the provider spec for the given provider ID or catalog alias.
func (r *ProviderRegistry) Get(id string) (ProviderSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id = strings.TrimSpace(id)
	if spec, ok := r.byID[id]; ok {
		return *spec, true
	}
	if alt := registryIDFromCatalogProvider(id); alt != id {
		if spec, ok := r.byID[alt]; ok {
			return *spec, true
		}
	}
	return ProviderSpec{}, false
}

// GetByEnv returns the provider spec for the given credential env var name.
func (r *ProviderRegistry) GetByEnv(env string) (ProviderSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	env = strings.TrimSpace(env)
	for _, s := range r.specs {
		if s.CredentialEnv == env {
			return s, true
		}
	}
	return ProviderSpec{}, false
}

// GetForLiveFetcher returns the provider spec for the given live fetcher key.
func (r *ProviderRegistry) GetForLiveFetcher(fetcherKey string) (ProviderSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.specs {
		if s.LiveFetcherKey == fetcherKey {
			return s, true
		}
	}
	return ProviderSpec{}, false
}

// All returns a copy of all registered provider specs.
func (r *ProviderRegistry) All() []ProviderSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ProviderSpec, len(r.specs))
	copy(out, r.specs)
	return out
}

// CredentialProviders returns all registered provider specs (every spec
// carries credential metadata; callers filter by RequiresKey as needed).
func (r *ProviderRegistry) CredentialProviders() []ProviderSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ProviderSpec, len(r.specs))
	copy(out, r.specs)
	return out
}

// LiveDiscoverable returns specs that have a live model-list fetcher registered.
func (r *ProviderRegistry) LiveDiscoverable() []ProviderSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []ProviderSpec
	for _, s := range r.specs {
		if s.LiveFetcherKey != "" {
			out = append(out, s)
		}
	}
	return out
}

// LiveFetcherKeys returns sorted live catalog provider keys that have fetchers.
func (r *ProviderRegistry) LiveFetcherKeys() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := map[string]bool{}
	var out []string
	for _, s := range r.specs {
		if s.LiveFetcherKey == "" || seen[s.LiveFetcherKey] {
			continue
		}
		seen[s.LiveFetcherKey] = true
		out = append(out, s.LiveFetcherKey)
	}
	sort.Strings(out)
	return out
}

// DeploymentEnvFallbacks derives seed env_fallbacks for registered deployments.
func (r *ProviderRegistry) DeploymentEnvFallbacks() map[string][]EnvFallback {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := map[string][]EnvFallback{}
	for _, s := range r.specs {
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
			} else {
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
