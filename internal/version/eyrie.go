// Package eyrie is the core LLM client library for hawk.
//
// It provides API provider configurations, model resolution, API limits,
// base types (messages, IDs, connectors), and error types.
//
// Sub-packages:
//   - types: Message types, content blocks, usage, IDs, connectors
//   - errors: Error message constants and utilities
//   - constants: API limits (image, PDF, media)
//   - catalog: Model catalog, tiers, names, deprecation, provider data
//   - config: Provider configuration, profiles, env, OpenAI-compatible runtime
//   - client: EyrieClient, factory, provider detection
//   - utils: Error utilities (SSL detection, API error sanitization)
package eyrie

import (
	_ "embed"
	"strings"

	"github.com/GrayCodeAI/eyrie/client"
)

// versionFile is the canonical version, embedded at compile time from the
// VERSION file at the repo root. The VERSION file is the single source of
// truth used by release tooling, CI, and the runtime Version variable.
//
//go:embed VERSION
var versionFile string

// Version of the eyrie library. Sourced from the VERSION file at the repo
// root — do not edit this variable directly. Bump VERSION instead, or let
// release-please/goreleaser do it.
var Version = strings.TrimSpace(versionFile)

func init() {
	// Propagate the canonical version into the client sub-package without
	// creating an import cycle (client cannot import eyrie).
	client.SetVersion(Version)
}
