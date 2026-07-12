package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog"
	"github.com/GrayCodeAI/eyrie/catalog/registry"
	"github.com/GrayCodeAI/eyrie/catalog/xiaomi"
	"github.com/GrayCodeAI/eyrie/catalog/zai"
	"github.com/GrayCodeAI/eyrie/config"
	"github.com/GrayCodeAI/eyrie/credentials"
	"github.com/GrayCodeAI/eyrie/runtime"
	"github.com/GrayCodeAI/eyrie/setup"
)

// SecretStoreName is safe presentation metadata for host UIs.
func SecretStoreName() string { return credentials.PlatformSecretStoreName() }

// CredentialStorageReport is safe to print and never includes secrets.
type CredentialStorageReport struct {
	PlatformStore string `json:"platform_store"`
	Writable      bool   `json:"writable"`
	Detail        string `json:"detail,omitempty"`
	Formatted     string `json:"formatted,omitempty"`
}

func CredentialStorage(ctx context.Context) CredentialStorageReport {
	ctx = nonNilContext(ctx)
	report := credentials.StorageReportFor(ctx)
	ok, detail := credentials.KeychainWriteAvailable(ctx)
	return CredentialStorageReport{
		PlatformStore: report.PlatformStore, Writable: ok, Detail: detail,
		Formatted: credentials.FormatStorageReport(report),
	}
}

func MigrateLegacyCredentials(ctx context.Context) (int, error) {
	return credentials.MigrateLegacyEnvFile(nonNilContext(ctx))
}

// HasCredentialEnv reports presence without exposing the credential value.
func (e *Engine) HasCredentialEnv(ctx context.Context, envVar string) bool {
	return e.hasCredential(nonNilContext(ctx), []string{envVar})
}

// SaveCredentialEnv validates and stores a legacy env-addressed credential.
// New callers should prefer SaveCredential(providerID, secret).
func (e *Engine) SaveCredentialEnv(ctx context.Context, envVar, secret string) error {
	envVar, secret = strings.TrimSpace(envVar), strings.TrimSpace(secret)
	if envVar == "" || secret == "" {
		return invalid("save_credential_env", "eyrie engine: credential env and value are required")
	}
	if err := config.ValidateCredentialSecret(envVar, secret); err != nil {
		return &Error{Code: ErrorInvalidRequest, Operation: "save_credential_env", Message: err.Error(), Cause: err}
	}
	if err := e.secretStore.Set(nonNilContext(ctx), credentials.AccountForEnv(envVar), secret); err != nil {
		return &Error{Code: ErrorInternal, Operation: "save_credential_env", Message: err.Error(), Cause: err}
	}
	return nil
}

func (e *Engine) CredentialEnvKeys(providerID string) []string {
	spec, ok := registry.SpecByProviderID(NormalizeProviderID(providerID))
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, envVar := range append(append([]string{spec.CredentialEnv}, spec.CredentialEnvFallbacks...), spec.CredentialAliases...) {
		envVar = strings.TrimSpace(envVar)
		if envVar != "" && !seen[envVar] {
			seen[envVar] = true
			out = append(out, envVar)
		}
	}
	return out
}

// GatewayRegion returns normalized region presentation for regional gateways.
func (e *Engine) GatewayRegion(providerID string) (label string, required bool) {
	cfg := config.LoadProviderConfig(e.providerConfigPath)
	providerID = NormalizeProviderID(providerID)
	switch providerID {
	case runtime.GatewayXiaomiTokenPlan:
		if cfg == nil {
			return "", true
		}
		region, err := xiaomi.NormalizeRegion(cfg.XiaomiMimoTokenPlanRegion)
		return string(region), err != nil
	case "zai_coding":
		if cfg == nil {
			return "", true
		}
		region, err := zai.NormalizeRegion(cfg.ZAICodingRegion)
		return string(region), err != nil
	default:
		return "", false
	}
}

// SetGatewayRegion persists regional gateway configuration at the Engine path.
func (e *Engine) SetGatewayRegion(ctx context.Context, providerID, value string) error {
	providerID = NormalizeProviderID(providerID)
	cfg := config.LoadProviderConfig(e.providerConfigPath)
	if cfg == nil {
		cfg = &config.ProviderConfig{}
	}
	switch providerID {
	case runtime.GatewayXiaomiTokenPlan:
		region, err := xiaomi.NormalizeRegion(value)
		if err != nil {
			return err
		}
		cfg.XiaomiMimoTokenPlanRegion = string(region)
		if base, err := config.ResolveXiaomiOpenAIBase(providerID, cfg); err == nil {
			cfg.XiaomiMimoTokenPlanBaseURL = base
		}
	case "zai_payg", "zai_coding":
		region, err := zai.NormalizeRegion(value)
		if err != nil {
			return err
		}
		if providerID == "zai_coding" {
			cfg.ZAICodingRegion = string(region)
			if base, err := zai.ResolveOpenAIBase(zai.PlanCoding, region, cfg.ZAICodingBaseURL); err == nil {
				cfg.ZAICodingBaseURL = base
			}
		} else {
			cfg.ZAIRegion = string(region)
			if base, err := zai.ResolveOpenAIBase(zai.PlanGeneral, region, cfg.ZAIBaseURL); err == nil {
				cfg.ZAIBaseURL = base
			}
		}
	default:
		return invalid("set_gateway_region", "eyrie engine: gateway does not support regions")
	}
	if err := config.SaveProviderConfig(cfg, e.providerConfigPath); err != nil {
		return err
	}
	e.ApplyGatewayEnvironment(ctx, providerID)
	return nil
}

// ApplyGatewayEnvironment derives process-local regional base URLs.
func (e *Engine) ApplyGatewayEnvironment(_ context.Context, providerID string) {
	cfg := config.LoadProviderConfig(e.providerConfigPath)
	if cfg == nil {
		return
	}
	switch NormalizeProviderID(providerID) {
	case runtime.GatewayXiaomiTokenPlan:
		if cfg.XiaomiMimoTokenPlanRegion != "" {
			_ = os.Setenv(config.EnvXiaomiTokenPlanRegion, cfg.XiaomiMimoTokenPlanRegion)
		}
		if cfg.XiaomiMimoTokenPlanBaseURL != "" {
			_ = os.Setenv(config.EnvXiaomiTokenPlanBaseURL, cfg.XiaomiMimoTokenPlanBaseURL)
		}
	case "zai_payg":
		if cfg.ZAIRegion != "" {
			_ = os.Setenv("ZAI_REGION", cfg.ZAIRegion)
		}
		if cfg.ZAIBaseURL != "" {
			_ = os.Setenv("ZAI_BASE_URL", cfg.ZAIBaseURL)
		}
	case "zai_coding":
		if cfg.ZAICodingRegion != "" {
			_ = os.Setenv("ZAI_CODING_REGION", cfg.ZAICodingRegion)
		}
		if cfg.ZAICodingBaseURL != "" {
			_ = os.Setenv("ZAI_CODING_BASE_URL", cfg.ZAICodingBaseURL)
		}
	}
}

type CatalogHealth struct {
	Path        string    `json:"path"`
	Exists      bool      `json:"exists"`
	ModifiedAt  time.Time `json:"modified_at,omitempty"`
	Size        int64     `json:"size,omitempty"`
	Models      int       `json:"models,omitempty"`
	Deployments int       `json:"deployments,omitempty"`
	Offerings   int       `json:"offerings,omitempty"`
	Stale       bool      `json:"stale,omitempty"`
	StaleAfter  time.Time `json:"stale_after,omitempty"`
	Source      string    `json:"source,omitempty"`
	Error       string    `json:"error,omitempty"`
}

func (e *Engine) CatalogHealth(ctx context.Context) CatalogHealth {
	health := CatalogHealth{Path: e.catalogPath}
	exists, modified, size, err := catalog.CacheInfo(e.catalogPath)
	health.Exists, health.ModifiedAt, health.Size = exists, modified, size
	if err != nil {
		health.Error = err.Error()
		return health
	}
	compiled, err := catalog.LoadCatalog(nonNilContext(ctx), catalog.LoadCatalogOptions{CachePath: e.catalogPath})
	if err != nil {
		health.Error = err.Error()
		return health
	}
	health.Models = len(compiled.ModelsByID)
	health.Deployments = len(compiled.DeploymentsByID)
	health.Offerings = len(compiled.OfferingsByID)
	if compiled.Catalog != nil {
		health.StaleAfter = compiled.Catalog.StaleAfter
		health.Stale = time.Now().UTC().After(compiled.Catalog.StaleAfter)
		if compiled.Catalog.Provenance != nil {
			health.Source = compiled.Catalog.Provenance.Source
		}
	}
	return health
}

func (e *Engine) CanonicalModel(ctx context.Context, modelID string) string {
	compiled, err := e.policyCatalog(ctx)
	if err == nil {
		if canonical, ok := compiled.CanonicalModelForAliasOrID(strings.TrimSpace(modelID)); ok {
			return canonical
		}
	}
	return strings.TrimSpace(modelID)
}

func (e *Engine) GatewayForModel(ctx context.Context, modelID string) string {
	compiled, _ := e.policyCatalog(ctx)
	return catalog.GatewayForModel(compiled, modelID)
}

func (e *Engine) DeploymentRoutingEnabled(override *bool) bool {
	if override != nil {
		return *override
	}
	cfg := config.LoadProviderConfig(e.providerConfigPath)
	return setup.UseDeploymentRouting(cfg)
}

func (e *Engine) DeploymentStatus(ctx context.Context, activeModel string) (string, error) {
	// Compatibility formatter remains Eyrie-owned while setup internals migrate
	// to injected paths. Host code receives presentation text only.
	report, err := setup.DeploymentStatusFromPaths(nonNilContext(ctx), activeModel, e.providerConfigPath, e.catalogPath)
	if err != nil {
		return "", err
	}
	return setup.FormatStatus(report), nil
}

type DeploymentSummary struct {
	RoutingSource string `json:"routing_source,omitempty"`
	RoutingStages int    `json:"routing_stages,omitempty"`
	Formatted     string `json:"formatted"`
}

func (e *Engine) DeploymentSummary(ctx context.Context, activeModel string) (DeploymentSummary, error) {
	report, err := setup.DeploymentStatusFromPaths(nonNilContext(ctx), activeModel, e.providerConfigPath, e.catalogPath)
	if err != nil {
		return DeploymentSummary{}, err
	}
	return DeploymentSummary{
		RoutingSource: report.RoutingSource, RoutingStages: report.RoutingStages,
		Formatted: setup.FormatStatus(report),
	}, nil
}

func (e *Engine) RoutingPreview(ctx context.Context, modelID string) (string, error) {
	return setup.RoutingPreviewFromPaths(nonNilContext(ctx), modelID, e.providerConfigPath, e.catalogPath)
}

// FormatSetupError returns a safe provider setup hint.
func FormatSetupError(providerID string, err error) string {
	if err == nil {
		return ""
	}
	return runtime.FormatSetupError(providerID, err).Error()
}

// CredentialGuidance returns provider-specific, non-secret setup guidance.
func CredentialGuidance(providerID, secret string) string {
	switch NormalizeProviderID(providerID) {
	case runtime.GatewayXiaomiTokenPlan:
		return xiaomi.KeyMismatchHint(xiaomi.BillingTokenPlan, secret)
	case xiaomi.ProviderPayAsYouGo:
		return xiaomi.KeyMismatchHint(xiaomi.BillingPayAsYouGo, secret)
	default:
		return ""
	}
}

func (e *Engine) DefaultProviderFilter(ctx context.Context) string {
	selection := e.EffectiveSelection(ctx, SelectionOptions{})
	return selection.Provider
}

func (e *Engine) Preflight(ctx context.Context) PreflightReport {
	ctx = nonNilContext(ctx)
	var checks []PreflightCheck
	health := e.CatalogHealth(ctx)
	if health.Error != "" || health.Models == 0 {
		checks = append(checks, PreflightCheck{Name: "catalog", Status: CheckWarn, Detail: fmt.Sprintf("catalog unavailable at %s", health.Path)})
	} else {
		checks = append(checks, PreflightCheck{Name: "catalog", Status: CheckOK, Detail: fmt.Sprintf("%d models cached at %s", health.Models, health.Path)})
	}
	storage := CredentialStorage(ctx)
	status := CheckWarn
	if storage.Writable {
		status = CheckOK
	}
	checks = append(checks, PreflightCheck{Name: "credentials_store", Status: status, Detail: storage.Detail})
	selection := e.EffectiveSelection(ctx, SelectionOptions{})
	if selection.HasConfiguredDeployment {
		checks = append(checks, PreflightCheck{Name: "credentials", Status: CheckOK, Detail: "at least one provider credential is configured in " + SecretStoreName()})
	} else {
		checks = append(checks, PreflightCheck{Name: "credentials", Status: CheckFail, Detail: "no provider credentials configured"})
	}
	if selection.Model == "" {
		checks = append(checks, PreflightCheck{Name: "model", Status: CheckFail, Detail: "no model selected"})
	} else {
		checks = append(checks, PreflightCheck{Name: "model", Status: CheckOK, Detail: selection.Model})
	}
	ready := true
	for _, check := range checks {
		if check.Status == CheckFail {
			ready = false
		}
	}
	return PreflightReport{Ready: ready, Checks: checks}
}

type CheckStatus string

const (
	CheckOK   CheckStatus = "ok"
	CheckWarn CheckStatus = "warn"
	CheckFail CheckStatus = "fail"
)

type PreflightCheck struct {
	Name   string      `json:"name"`
	Status CheckStatus `json:"status"`
	Detail string      `json:"detail"`
}

type PreflightReport struct {
	Ready  bool             `json:"ready"`
	Checks []PreflightCheck `json:"checks"`
}

func FormatPreflight(report PreflightReport) string {
	var b strings.Builder
	if report.Ready {
		b.WriteString("Preflight: ready to chat\n")
	} else {
		b.WriteString("Preflight: setup incomplete\n")
	}
	for _, check := range report.Checks {
		icon := "+"
		if check.Status == CheckWarn {
			icon = "!"
		} else if check.Status == CheckFail {
			icon = "x"
		}
		fmt.Fprintf(&b, "  %s %s: %s\n", icon, check.Name, check.Detail)
	}
	return strings.TrimRight(b.String(), "\n")
}

type ProviderStateSecurity struct {
	Path       string `json:"path"`
	HasSecrets bool   `json:"has_secrets"`
	Detail     string `json:"detail,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ProviderStateSecurityStatus inspects provider.json without returning values.
func (e *Engine) ProviderStateSecurityStatus() ProviderStateSecurity {
	status := ProviderStateSecurity{Path: e.providerConfigPath}
	cfg, err := config.LoadProviderConfigWithError(e.providerConfigPath)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	if cfg == nil {
		return status
	}
	for id, deployment := range cfg.Deployments {
		if deploymentContainsSecrets(deployment) {
			status.HasSecrets = true
			status.Detail = "deployment " + id + " has secret fields on disk"
			return status
		}
	}
	return status
}

// MigrateProviderSecrets atomically strips historical secret fields from
// provider.json while preserving routing and non-secret deployment metadata.
func (e *Engine) MigrateProviderSecrets() error {
	path := e.providerConfigPath
	marker, backup := path+".secrets-migrated", path+".pre-secret-migrate.bak"
	if _, err := os.Stat(marker); err == nil {
		_ = os.Remove(backup)
		return nil
	}
	data, err := os.ReadFile(path) // #nosec G304 -- engine-owned state path
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var cfg config.ProviderConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	changed := false
	for id, deployment := range cfg.Deployments {
		if deploymentContainsSecrets(deployment) {
			changed = true
		}
		cfg.Deployments[id] = config.SanitizeDeploymentConfigForDisk(deployment)
	}
	if !changed {
		return os.WriteFile(marker, []byte("ok\n"), 0o600)
	}
	if err := os.WriteFile(backup, data, 0o600); err != nil {
		return err
	}
	if err := writeProviderConfigAtomic(path, &cfg); err != nil {
		return err
	}
	if err := os.WriteFile(marker, []byte("ok\n"), 0o600); err != nil {
		return err
	}
	return os.Remove(backup)
}

func writeProviderConfigAtomic(path string, cfg *config.ProviderConfig) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".provider-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func deploymentContainsSecrets(deployment config.DeploymentConfig) bool {
	return strings.TrimSpace(deployment.APIKey) != "" || strings.TrimSpace(deployment.Token) != "" ||
		strings.TrimSpace(deployment.SecretAccessKey) != "" || strings.TrimSpace(deployment.AccessKeyID) != "" ||
		strings.TrimSpace(deployment.SessionToken) != ""
}
