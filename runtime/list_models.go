package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/live"
	"github.com/GrayCodeAI/eyrie/catalog/registry"
	eyriecfg "github.com/GrayCodeAI/eyrie/config"
)

// ListModelSource selects cache vs live model listing.
type ListModelSource string

const (
	ListSourceAuto  ListModelSource = "auto"
	ListSourceCache ListModelSource = "cache"
	ListSourceLive  ListModelSource = "live"
)

// ListModelsOpts configures unified model listing for host UIs.
type ListModelsOpts struct {
	ProviderID string
	Source     ListModelSource
	Refresh    bool
}

// ModelEntry is one row for host model pickers.
type ModelEntry struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	ProviderID  string `json:"provider_id"`
	Source      string `json:"source"`
	Installed   bool   `json:"installed,omitempty"`
}

// ListModels returns models for a provider using registry-driven source selection.
func ListModels(ctx context.Context, opts ListModelsOpts) ([]ModelEntry, error) {
	providerID := strings.TrimSpace(opts.ProviderID)
	if providerID == "" {
		return nil, fmt.Errorf("runtime: provider required")
	}
	spec, ok := registry.SpecByProviderID(providerID)
	if !ok {
		return listModelsFromCache(ctx, providerID, "cache")
	}
	if opts.Refresh {
		if _, err := Discover(ctx); err != nil {
			return nil, err
		}
	}
	switch opts.Source {
	case ListSourceLive:
		return listModelsLive(ctx, spec)
	case ListSourceCache:
		return listModelsFromCache(ctx, providerID, "cache")
	default:
		return listModelsAuto(ctx, spec)
	}
}

func listModelsAuto(ctx context.Context, spec registry.ProviderSpec) ([]ModelEntry, error) {
	switch spec.ModelStrategy {
	case registry.StrategyLiveOnly:
		return listModelsLive(ctx, spec)
	default:
		cached, err := listModelsFromCache(ctx, spec.ProviderID, "cache")
		if err != nil {
			return nil, err
		}
		if len(cached) > 0 {
			return cached, nil
		}
		return listModelsLive(ctx, spec)
	}
}

func listModelsFromCache(ctx context.Context, providerID, source string) ([]ModelEntry, error) {
	rt, err := Load(ctx)
	if err != nil {
		return nil, err
	}
	entries := rt.ModelEntriesForProvider(providerID)
	return entriesToModelList(entries, providerID, source, false), nil
}

func listModelsLive(ctx context.Context, spec registry.ProviderSpec) ([]ModelEntry, error) {
	if spec.LiveFetcherKey == "" {
		return listModelsFromCache(ctx, spec.ProviderID, "cache")
	}
	env := eyriecfg.DiscoveryEnvMap(ctx)
	if spec.ProviderID == "ollama" && strings.TrimSpace(env["OLLAMA_BASE_URL"]) == "" {
		env = copyEnvMap(env)
		env["OLLAMA_BASE_URL"] = eyriecfg.OllamaDefaultBaseURL
	}
	entries, err := live.Fetch(spec.LiveFetcherKey, env)
	if err != nil {
		return nil, FormatSetupError(spec.ProviderID, err)
	}
	if len(entries) == 0 && spec.ModelStrategy == registry.StrategyLiveOnly {
		if spec.ProviderID == "ollama" {
			return nil, FormatSetupError("ollama", fmt.Errorf("ollama is running but no models are installed — run: ollama pull llama3.2"))
		}
		return nil, fmt.Errorf("runtime: no live models returned for %s", spec.ProviderID)
	}
	installed := spec.ModelStrategy == registry.StrategyLiveOnly
	return liveEntriesToModelList(entries, spec.ProviderID, "live", installed), nil
}

func liveEntriesToModelList(entries []live.Entry, providerID, source string, installed bool) []ModelEntry {
	catalogEntries := make([]catalog.ModelCatalogEntry, len(entries))
	for i, e := range entries {
		catalogEntries[i] = catalog.ModelCatalogEntry{
			ID: e.ID, DisplayName: e.DisplayName, ContextWindow: e.ContextWindow, MaxOutput: e.MaxOutput,
		}
	}
	return entriesToModelList(catalogEntries, providerID, source, installed)
}

func entriesToModelList(entries []catalog.ModelCatalogEntry, providerID, source string, installed bool) []ModelEntry {
	out := make([]ModelEntry, 0, len(entries))
	seen := map[string]bool{}
	for _, e := range entries {
		id := strings.TrimSpace(e.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		label := strings.TrimSpace(e.DisplayName)
		if label == "" {
			label = id
		}
		out = append(out, ModelEntry{
			ID:          id,
			DisplayName: label,
			ProviderID:  providerID,
			Source:      source,
			Installed:   installed,
		})
	}
	return out
}

func copyEnvMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
}

// FormatSetupError maps provider setup failures to user-facing hints.
func FormatSetupError(providerID string, err error) error {
	if err == nil {
		return nil
	}
	if strings.TrimSpace(providerID) == "ollama" {
		return eyriecfg.FormatOllamaConnectError(err)
	}
	return err
}

// ListProviderSetupOptions returns hub rows for host /config UIs.
func ListProviderSetupOptions(ctx context.Context) []ProviderSetupOption {
	_ = ctx
	var out []ProviderSetupOption
	st := eyriecfg.DiscoveryEnvMap(ctx)
	hasAny := eyriecfg.HasAnyConfiguredDeployment(ctx)
	if hasAny {
		out = append(out, ProviderSetupOption{Action: "model", Label: "Pick model"})
	}
	out = append(
		out,
		ProviderSetupOption{Action: "apikey", Label: "Paste API key"},
		ProviderSetupOption{Action: "ollama", Label: "Ollama (local — no key)"},
	)
	_ = st
	return out
}

// ProviderSetupOption is one hub row in host /config.
type ProviderSetupOption struct {
	Action string `json:"action"`
	Label  string `json:"label"`
}
