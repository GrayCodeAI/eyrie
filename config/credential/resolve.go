package credential

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

// CredentialProviderOption is one row for host provider pickers (JSON-safe).
type CredentialProviderOption struct {
	ProviderID   string `json:"provider_id"`
	DeploymentID string `json:"deployment_id"`
	EnvVar       string `json:"env_var"`
	DisplayName  string `json:"display_name"`
	Inferred     bool   `json:"inferred"`
	RequiresKey  bool   `json:"requires_key"`
	Rank         int    `json:"rank"`
}

// CredentialResolveResult is returned after paste-key format validation + provider listing.
type CredentialResolveResult struct {
	FormatOK    bool                       `json:"format_ok"`
	FormatError string                     `json:"format_error,omitempty"`
	Providers   []CredentialProviderOption `json:"providers"`
	// ProbeDisambiguationUsed is true when ambiguous keys were verified via live provider probes.
	ProbeDisambiguationUsed bool `json:"probe_disambiguation_used,omitempty"`
}

// ValidateKeyFormat checks a pasted secret before any provider is chosen.
func ValidateKeyFormat(secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return errEmptyCredential
	}
	if LooksLikePlaceholderSecret(secret) {
		return errPlaceholderCredential
	}
	if len(secret) < 8 {
		return errCredentialTooShort
	}
	return nil
}

// ListCredentialProviders returns all registered API-key providers for host pickers.
func ListCredentialProviders() []CredentialProviderOption {
	specs := registry.DefaultRegistry.CredentialProviders()
	out := make([]CredentialProviderOption, len(specs))
	for i, spec := range specs {
		out[i] = optionFromSpec(spec, false, spec.SortOrder)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rank < out[j].Rank })
	return out
}

// ResolveCredential validates format and returns all providers with inferred matches ranked first.
func ResolveCredential(ctx context.Context, secret string) CredentialResolveResult {
	secret = strings.TrimSpace(secret)
	if err := ValidateKeyFormat(secret); err != nil {
		return CredentialResolveResult{FormatOK: false, FormatError: err.Error()}
	}
	matches := rankedProviderMatches(secret)
	inferredSet := inferredSetFromMatches(matches)
	var out []CredentialProviderOption
	for _, spec := range registry.DefaultRegistry.CredentialProviders() {
		if !spec.RequiresKey {
			continue
		}
		rank, inf := inferredSet[spec.ProviderID]
		if inf {
			rank = spec.SortOrder*100 + rank
		} else {
			rank = spec.SortOrder + 1000
		}
		out = append(out, optionFromSpec(spec, inf, rank))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Inferred != out[j].Inferred {
			return out[i].Inferred
		}
		if out[i].Rank != out[j].Rank {
			return out[i].Rank < out[j].Rank
		}
		return out[i].DisplayName < out[j].DisplayName
	})
	var probeUsed bool
	out, probeUsed = applyProbeDisambiguation(ctx, secret, out)
	return CredentialResolveResult{FormatOK: true, Providers: out, ProbeDisambiguationUsed: probeUsed}
}

// InferenceFromOption converts a picker row back to persistence metadata.
func InferenceFromOption(opt CredentialProviderOption) CredentialInference {
	return CredentialInference{
		ProviderID:   opt.ProviderID,
		DeploymentID: opt.DeploymentID,
		EnvVar:       opt.EnvVar,
		DisplayName:  opt.DisplayName,
	}
}

func optionFromSpec(spec registry.ProviderSpec, inferred bool, rank int) CredentialProviderOption {
	return CredentialProviderOption{
		ProviderID:   spec.ProviderID,
		DeploymentID: spec.DeploymentID,
		EnvVar:       spec.CredentialEnv,
		DisplayName:  spec.DisplayName,
		Inferred:     inferred,
		RequiresKey:  spec.RequiresKey,
		Rank:         rank,
	}
}

// LocalCredentialInference returns setup metadata for no-key providers (e.g. Ollama).
func LocalCredentialInference(providerID string) (CredentialInference, error) {
	spec, ok := registry.DefaultRegistry.Get(strings.TrimSpace(providerID))
	if !ok || spec.RequiresKey {
		return CredentialInference{}, fmt.Errorf("local credential: unknown provider %q", providerID)
	}
	return CredentialInference{
		ProviderID:   spec.ProviderID,
		DeploymentID: spec.DeploymentID,
		EnvVar:       spec.CredentialEnv,
		DisplayName:  spec.DisplayName,
	}, nil
}

// matchedProviderIDsFromRegistry returns provider IDs ranked by pattern match (registry + learned).
func matchedProviderIDsFromRegistry(secret string) []string {
	matches := rankedProviderMatches(secret)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if m.inferred {
			out = append(out, m.providerID)
		}
	}
	return out
}
