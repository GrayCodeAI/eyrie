package engine

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GrayCodeAI/eyrie/catalog"
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
	for _, provider := range e.CredentialProviders(context.Background()) {
		if NormalizeProviderID(provider.ProviderID) == NormalizeProviderID(providerID) {
			return []string{provider.EnvVar}
		}
	}
	return nil
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
	report, err := setup.DeploymentStatus(nonNilContext(ctx), activeModel)
	if err != nil {
		return "", err
	}
	return setup.FormatStatus(report), nil
}

func (e *Engine) RoutingPreview(ctx context.Context, modelID string) (string, error) {
	return setup.RoutingPreview(nonNilContext(ctx), modelID)
}

// FormatSetupError returns a safe provider setup hint.
func FormatSetupError(providerID string, err error) string {
	if err == nil {
		return ""
	}
	return runtime.FormatSetupError(providerID, err).Error()
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
