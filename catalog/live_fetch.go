package catalog

import (
	"strings"
	"time"
)

// liveDiscoverableDeployments lists deployments enriched via live provider list APIs during discover.
var liveDiscoverableDeployments = map[string]func(map[string]string) ([]ModelCatalogEntry, error){
	"openrouter":  fetchOpenRouterCatalog,
	"canopywave":  fetchCanopyWaveCatalog,
}

// LiveDiscoverableDeploymentIDs returns deployment IDs with live model-list APIs.
func LiveDiscoverableDeploymentIDs() []string {
	ids := make([]string, 0, len(liveDiscoverableDeployments))
	for id := range liveDiscoverableDeployments {
		ids = append(ids, id)
	}
	return ids
}

func apiKeyPresent(env map[string]string, deploymentID string) bool {
	for _, key := range EnvVarsForDeployment(deploymentID) {
		if strings.Contains(strings.ToLower(key), "api_key") && strings.TrimSpace(env[key]) != "" {
			return true
		}
	}
	// Also accept common *_API_KEY names when env_fallbacks not yet on catalog.
	switch deploymentID {
	case "openrouter":
		return strings.TrimSpace(env["OPENROUTER_API_KEY"]) != ""
	case "canopywave":
		return strings.TrimSpace(env["CANOPYWAVE_API_KEY"]) != ""
	default:
		return false
	}
}

func fetchLiveProviderCatalog(env map[string]string) (ModelCatalog, []LiveProviderEnrichment) {
	cat := ModelCatalog{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Source:    "live-providers",
		Providers: make(map[string][]ModelCatalogEntry),
	}
	var enrichment []LiveProviderEnrichment
	for deploymentID, fetch := range liveDiscoverableDeployments {
		if !apiKeyPresent(env, deploymentID) {
			continue
		}
		models, err := fetch(env)
		if err != nil {
			enrichment = append(enrichment, LiveProviderEnrichment{Provider: deploymentID, Error: err.Error()})
			continue
		}
		if len(models) == 0 {
			enrichment = append(enrichment, LiveProviderEnrichment{Provider: deploymentID, Error: "no models returned"})
			continue
		}
		cat.Providers[deploymentID] = models
		enrichment = append(enrichment, LiveProviderEnrichment{Provider: deploymentID, ModelCount: len(models)})
	}
	return cat, enrichment
}
