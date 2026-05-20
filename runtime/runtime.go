// Package runtime is the only stable API surface for host applications (e.g. hawk).
// Import github.com/GrayCodeAI/eyrie/runtime — not catalog/setup/config directly.
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/client"
	"github.com/GrayCodeAI/eyrie/config"
	"github.com/GrayCodeAI/eyrie/credentials"
	"github.com/GrayCodeAI/eyrie/setup"
)

// Runtime is a loaded eyrie control plane: catalog cache + routing + env-backed credentials.
type Runtime struct {
	Catalog      *catalog.CompiledCatalogV1
	Provider     *config.ProviderConfig
	ProviderPath string
}

// Apply discovers the model catalog and writes ~/.eyrie/provider.json (routing only; secrets stay in env).
func Apply(ctx context.Context, creds catalog.Credentials) (*ApplyResult, error) {
	result, err := setup.ApplyCredentials(ctx, creds)
	if err != nil {
		return nil, err
	}
	return &ApplyResult{
		Catalog:      result.Catalog,
		Provider:     result.ProviderConfig,
		ProviderPath: result.ProviderConfigPath,
		RoutingJSON:  result.RoutingJSON,
		Setup:        result.Setup,
	}, nil
}

// ApplyResult summarizes an Apply call.
type ApplyResult struct {
	Catalog      *catalog.RefreshResult
	Provider     *config.ProviderConfig
	ProviderPath string
	RoutingJSON  string
	Setup        *setup.SetupUI
}

// SetCredential stores a provider secret (env var name + value). Never log the secret argument.
func SetCredential(ctx context.Context, envKey, secret string) error {
	envKey = strings.TrimSpace(envKey)
	secret = strings.TrimSpace(secret)
	if envKey == "" || secret == "" {
		return fmt.Errorf("runtime: env key and secret required")
	}
	if err := credentials.DefaultStore().Set(ctx, credentials.AccountForEnv(envKey), secret); err != nil {
		return fmt.Errorf("runtime: save credential: %w", err)
	}
	// Secrets stay in the store only — not in process env (agents / printenv cannot read them).
	return nil
}

// Load reads the on-disk catalog and provider config without network refresh.
func Load(ctx context.Context) (*Runtime, error) {
	compiled, err := setup.LoadCompiledCatalog(ctx)
	if err != nil {
		return nil, err
	}
	cfg := config.LoadProviderConfig("")
	if cfg != nil {
		cfg = config.EnsureDeploymentConfigV2(cfg)
	}
	return &Runtime{
		Catalog:      compiled,
		Provider:     cfg,
		ProviderPath: config.GetProviderConfigPath(),
	}, nil
}

// Discover runs a full catalog refresh then reloads runtime state.
func Discover(ctx context.Context) (*ApplyResult, error) {
	return Apply(ctx, config.DiscoveryCredentials(ctx))
}

// ChatProvider builds the LLM client (deployment router when configured).
func (r *Runtime) ChatProvider(ctx context.Context) (client.Provider, error) {
	cfg := r.Provider
	if cfg == nil {
		cfg = config.LoadProviderConfig("")
	}
	p, err := setup.DeploymentProvider(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// RoutingPreviewJSON returns effective routing for a model ID.
func (r *Runtime) RoutingPreviewJSON(model string) (string, error) {
	return setup.RoutingPreview(ctxWithBackground(), model)
}

func ctxWithBackground() context.Context {
	return context.Background()
}

// ProviderConfigJSON returns provider.json as indented JSON.
func (r *Runtime) ProviderConfigJSON() (string, error) {
	cfg := r.Provider
	if cfg == nil {
		return "{}", nil
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ModelIDs returns sorted canonical model IDs from the catalog.
func (r *Runtime) ModelIDs() []string {
	if r == nil || r.Catalog == nil {
		return nil
	}
	out := make([]string, 0, len(r.Catalog.ModelsByID))
	for id := range r.Catalog.ModelsByID {
		out = append(out, id)
	}
	return out
}

// DeploymentRows lists deployments with credential status from env (not from provider.json secrets).
func (r *Runtime) DeploymentRows() ([]DeploymentRow, error) {
	if r == nil || r.Catalog == nil {
		return nil, fmt.Errorf("runtime: catalog not loaded")
	}
	cfg := r.Provider
	if cfg == nil {
		cfg = &config.ProviderConfig{}
	}
	configured := setup.ConfiguredDeployments(cfg)
	var out []DeploymentRow
	for id, dep := range r.Catalog.DeploymentsByID {
		row := DeploymentRow{
			ID:         id,
			Name:       dep.Name,
			ProviderID: dep.ProviderID,
		}
		if _, ok := configured[id]; ok {
			if _, live := setup.ProviderForDeployment(id, configured[id]); live {
				row.Configured = true
				row.Status = "ready"
			} else {
				row.Status = "incomplete"
			}
		} else {
			row.Status = "needs credentials"
		}
		out = append(out, row)
	}
	return out, nil
}

// DeploymentRow is a deployment plus env credential status.
type DeploymentRow struct {
	ID         string
	Name       string
	ProviderID string
	Configured bool
	Status     string
	PrimaryEnv string
}

// CredentialTargets lists provider-facing API key env vars for simple UIs.
func (r *Runtime) CredentialTargets() []CredentialTarget {
	compiled := r.Catalog
	if compiled == nil {
		bootstrap := catalog.BootstrapCatalogV1()
		c, err := catalog.CompileCatalogV1(&bootstrap)
		if err != nil {
			return nil
		}
		compiled = c
	}
	seen := map[string]bool{}
	var out []CredentialTarget
	for _, depID := range catalog.ProviderIDsFromCompiled(compiled) {
		for _, id := range configuredDeploymentIDsForProvider(compiled, depID) {
			env := catalog.PrimaryAPIKeyEnvForDeployment(compiled, id)
			if env == "" || seen[env] {
				continue
			}
			seen[env] = true
			out = append(out, CredentialTarget{
				ProviderID:   depID,
				DeploymentID: id,
				EnvVar:       env,
				Set:          credentials.HasSecret(context.Background(), env),
			})
		}
	}
	return out
}

// CredentialTarget is one API key slot in a host /config UI.
type CredentialTarget struct {
	ProviderID   string
	DeploymentID string
	EnvVar       string
	Set          bool
}

func configuredDeploymentIDsForProvider(compiled *catalog.CompiledCatalogV1, providerID string) []string {
	if compiled == nil || compiled.Catalog == nil {
		return nil
	}
	var out []string
	for id, dep := range compiled.Catalog.Deployments {
		if catalog.CanonicalProviderID(dep.ProviderID) == catalog.CanonicalProviderID(providerID) {
			out = append(out, id)
		}
	}
	return out
}

// DefaultPaths reports standard eyrie paths on disk.
func DefaultPaths() (catalogPath, providerPath string) {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".eyrie", "model_catalog.json"),
		config.GetProviderConfigPath()
}
