package catalog

import (
	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

// CredentialProviderSpec defines paste-key setup metadata (derived from registry).
type CredentialProviderSpec struct {
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

// CredentialProviderRegistry is derived from catalog/registry (single source of truth).
var CredentialProviderRegistry = deriveCredentialRegistry()

func deriveCredentialRegistry() []CredentialProviderSpec {
	rows := registry.CredentialRegistry()
	out := make([]CredentialProviderSpec, len(rows))
	for i, r := range rows {
		out[i] = CredentialProviderSpec{
			ProviderID:   r.ProviderID,
			DisplayName:  r.DisplayName,
			DeploymentID: r.DeploymentID,
			EnvVar:       r.EnvVar,
			KeyPrefixes:  r.KeyPrefixes,
			ProbeKind:    r.ProbeKind,
			ProbeBaseURL: r.ProbeBaseURL,
			RequiresKey:  r.RequiresKey,
			SortOrder:    r.SortOrder,
		}
	}
	return out
}

// EnsureCredentialRegistryInCatalog merges registry providers/deployments into catalog v1.
func EnsureCredentialRegistryInCatalog(c *CatalogV1) {
	if c == nil {
		return
	}
	if c.Providers == nil {
		c.Providers = map[string]ProviderV1{}
	}
	if c.Deployments == nil {
		c.Deployments = map[string]DeploymentV1{}
	}
	for _, spec := range registry.All() {
		pid := CanonicalProviderID(spec.ProviderID)
		if c.Providers[pid].ID == "" {
			c.Providers[pid] = ProviderV1{ID: pid, Name: spec.DisplayName}
		}
		if c.Deployments[spec.DeploymentID].ID == "" {
			c.Deployments[spec.DeploymentID] = DeploymentV1{
				ID:                  spec.DeploymentID,
				Name:                spec.DisplayName,
				ProviderID:          pid,
				APIProtocolID:       spec.APIProtocolID,
				AdapterConstructor:  spec.AdapterID,
				NativeModelIDSource: NativeModelIDDiscovered,
			}
		}
	}
	EnsureDeploymentEnvFallbacks(c)
}

func SpecByEnvVar(env string) (CredentialProviderSpec, bool) {
	for _, s := range CredentialProviderRegistry {
		if s.EnvVar == env {
			return s, true
		}
	}
	return CredentialProviderSpec{}, false
}

func SpecByProviderID(id string) (CredentialProviderSpec, bool) {
	id = CanonicalProviderID(id)
	for _, s := range CredentialProviderRegistry {
		if CanonicalProviderID(s.ProviderID) == id {
			return s, true
		}
	}
	return CredentialProviderSpec{}, false
}

// ProviderDisplayName returns UI label from registry.
func ProviderDisplayName(providerID string) string {
	return registry.DisplayName(providerID)
}
