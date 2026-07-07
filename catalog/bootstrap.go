package catalog

import "time"

const bootstrapSource = "bootstrap"

// BootstrapSource returns the provenance label for the embedded catalog seed.
func BootstrapSource() string {
	return bootstrapSource
}

// BootstrapCatalog returns deployment/provider wiring only — no chat models.
// Chat models come from the published catalog cache and live provider discovery.
func BootstrapCatalog() Catalog {
	generatedAt := time.Now().UTC().Truncate(time.Second)
	c := Catalog{
		SchemaVersion: CatalogSchemaVersion,
		GeneratedAt:   generatedAt,
		StaleAfter:    generatedAt.Add(24 * time.Hour),
		Providers:     defaultProviders(),
		Protocols:     defaultProtocols(),
		Deployments:   defaultDeployments(),
		Models:        map[string]Model{},
		Aliases:       map[string]string{},
		Offerings:     nil,
		Provenance:    &Provenance{Source: bootstrapSource, ObservedAt: generatedAt},
	}
	EnsureDeploymentEnvFallbacks(&c)
	EnsureCredentialRegistryInCatalog(&c)
	return c
}

// IsBootstrapCatalog reports whether c is the empty wiring-only catalog.
func IsBootstrapCatalog(c *Catalog) bool {
	return c != nil && c.Provenance != nil && c.Provenance.Source == bootstrapSource
}
