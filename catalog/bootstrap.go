package catalog

import "time"

const bootstrapSource = "bootstrap"

// BootstrapSource returns the provenance label for the embedded catalog seed.
func BootstrapSource() string {
	return bootstrapSource
}

// BootstrapCatalogV1 returns deployment/provider wiring only — no chat models.
// Chat models come from the published catalog cache and live provider discovery.
func BootstrapCatalogV1() CatalogV1 {
	generatedAt := time.Now().UTC().Truncate(time.Second)
	c := CatalogV1{
		SchemaVersion: CatalogV1SchemaVersion,
		GeneratedAt:   generatedAt,
		StaleAfter:    generatedAt.Add(24 * time.Hour),
		Providers:     defaultProvidersV1(),
		APIProtocols:  defaultAPIProtocolsV1(),
		Deployments:   defaultDeploymentsV1(),
		Models:        map[string]ModelV1{},
		Aliases:       map[string]string{},
		Offerings:     nil,
		Provenance:    &CatalogProvenanceV1{Source: bootstrapSource, ObservedAt: generatedAt},
	}
	EnsureDeploymentEnvFallbacks(&c)
	EnsureCredentialRegistryInCatalog(&c)
	return c
}

// IsBootstrapCatalog reports whether c is the empty wiring-only catalog.
func IsBootstrapCatalog(c *CatalogV1) bool {
	return c != nil && c.Provenance != nil && c.Provenance.Source == bootstrapSource
}
