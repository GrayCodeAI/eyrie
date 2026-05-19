package config

import (
	"context"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/credentials"
)

// DiscoveryEnvFromOS reads credential env vars defined by the eyrie catalog
// (deployment env_fallbacks). Falls back to legacy runtime profiles only when the
// catalog defines no env keys (e.g. empty cache before first refresh).
func DiscoveryEnvFromOS() map[string]string {
	ctx := context.Background()
	compiled, err := catalog.LoadCatalogForDiscovery(ctx)
	if err == nil && compiled != nil {
		keys := catalog.DiscoveryEnvKeysFromCatalog(compiled)
		if len(keys) > 0 {
			return catalog.ReadOSEnv(keys)
		}
	}
	return discoveryEnvFromProfilesLegacy()
}

// DiscoveryCredentialsFromOS builds catalog credentials from the process environment.
func DiscoveryCredentialsFromOS() catalog.Credentials {
	return DiscoveryCredentials(context.Background())
}

// DiscoveryCredentials merges keychain/env store with the process environment.
func DiscoveryCredentials(ctx context.Context) catalog.Credentials {
	credentials.ApplyToProcess(ctx, credentials.DefaultStore())
	env := filterPlaceholderEnv(DiscoveryEnvFromOS())
	for k, v := range credentials.APIKeysMap(ctx, credentials.DefaultStore()) {
		if strings.TrimSpace(v) != "" && !LooksLikePlaceholderSecret(v) {
			env[k] = v
		}
	}
	return catalog.Credentials{APIKeys: env}
}

func filterPlaceholderEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return env
	}
	out := make(map[string]string, len(env))
	for k, v := range env {
		if strings.TrimSpace(v) != "" && !LooksLikePlaceholderSecret(v) {
			out[k] = v
		}
	}
	return out
}

// discoveryEnvFromProfilesLegacy is deprecated: use catalog deployment env_fallbacks
// from the published catalog instead. Kept only when catalog has no credential metadata yet.
func discoveryEnvFromProfilesLegacy() map[string]string {
	keys := discoveryEnvKeysFromProfiles()
	return catalog.ReadOSEnv(keys)
}

func discoveryEnvKeysFromProfiles() []string {
	seen := map[string]bool{}
	var keys []string
	add := func(k string) {
		if k == "" || seen[k] {
			return
		}
		seen[k] = true
		keys = append(keys, k)
	}
	collect := func(p RuntimeProviderProfile) {
		for _, d := range p.APIKeys {
			add(d.Env)
		}
		for _, k := range p.DetectionEnv {
			add(k)
		}
		for _, k := range p.BaseURLEnv {
			add(k)
		}
	}
	for _, p := range OpenAICompatibleRuntimeProfiles {
		collect(p)
	}
	add("OLLAMA_BASE_URL")
	add("OLLAMA_API_KEY")
	return keys
}
