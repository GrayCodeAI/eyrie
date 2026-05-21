package config

import (
	"context"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/credentials"
)

// DiscoveryCredentials loads API keys from the OS secret store (not process env or .env files).
func DiscoveryCredentials(ctx context.Context) catalog.Credentials {
	if ctx == nil {
		ctx = context.Background()
	}
	env := filterPlaceholderEnv(credentials.APIKeysMap(ctx, credentials.DefaultStore()))
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
