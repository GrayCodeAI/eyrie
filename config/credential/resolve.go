package credential

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
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
	out := make([]CredentialProviderOption, len(catalog.CredentialProviderRegistry))
	for i, spec := range catalog.CredentialProviderRegistry {
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
	inferredSet := map[string]int{}
	for rank, pid := range matchedProviderIDsFromRegistry(secret) {
		inferredSet[pid] = rank
	}
	var out []CredentialProviderOption
	for _, spec := range catalog.CredentialProviderRegistry {
		if !spec.RequiresKey {
			continue
		}
		rank, inf := inferredSet[spec.ProviderID]
		if !inf {
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
	return CredentialResolveResult{FormatOK: true, Providers: out}
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

func optionFromSpec(spec catalog.CredentialProviderSpec, inferred bool, rank int) CredentialProviderOption {
	return CredentialProviderOption{
		ProviderID:   spec.ProviderID,
		DeploymentID: spec.DeploymentID,
		EnvVar:       spec.EnvVar,
		DisplayName:  spec.DisplayName,
		Inferred:     inferred,
		RequiresKey:  spec.RequiresKey,
		Rank:         rank,
	}
}

// LocalCredentialInference returns setup metadata for no-key providers (e.g. Ollama).
func LocalCredentialInference(providerID string) (CredentialInference, error) {
	spec, ok := catalog.SpecByProviderID(strings.TrimSpace(providerID))
	if !ok || spec.RequiresKey {
		return CredentialInference{}, fmt.Errorf("local credential: unknown provider %q", providerID)
	}
	return CredentialInference{
		ProviderID:   spec.ProviderID,
		DeploymentID: spec.DeploymentID,
		EnvVar:       spec.EnvVar,
		DisplayName:  spec.DisplayName,
	}, nil
}

func matchedProviderIDsFromRegistry(secret string) []string {
	type match struct {
		id     string
		prefix string
		rank   int
	}
	var matches []match
	for _, spec := range catalog.CredentialProviderRegistry {
		for pi, prefix := range spec.KeyPrefixes {
			if prefix != "" && strings.HasPrefix(secret, prefix) {
				matches = append(matches, match{id: spec.ProviderID, prefix: prefix, rank: spec.SortOrder*100 + pi})
			}
		}
	}
	if len(matches) == 0 {
		return nil
	}
	sort.Slice(matches, func(i, j int) bool {
		if len(matches[i].prefix) != len(matches[j].prefix) {
			return len(matches[i].prefix) > len(matches[j].prefix)
		}
		return matches[i].rank < matches[j].rank
	})
	seen := map[string]bool{}
	var out []string
	for _, m := range matches {
		if seen[m.id] {
			continue
		}
		seen[m.id] = true
		out = append(out, m.id)
	}
	return out
}
