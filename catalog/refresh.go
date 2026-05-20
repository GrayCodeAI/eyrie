package catalog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RefreshResult summarizes a strict remote catalog refresh (eyrie published catalog).
type RefreshResult struct {
	Compiled   *CompiledCatalogV1
	CachePath  string
	Source     string // remote, cache, embedded, remote+providers
	RemoteURL  string
	Refreshed  bool
	StaleAfter time.Time
	// RemoteRefreshed is true when the published remote catalog was fetched successfully.
	RemoteRefreshed bool
	// LiveProviders lists provider APIs queried with API keys (empty if none attempted).
	LiveProviders []LiveProviderEnrichment
}

// DefaultCachePath returns the shared model catalog cache location.
func DefaultCachePath() string {
	if p := strings.TrimSpace(os.Getenv("EYRIE_MODEL_CATALOG_PATH")); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".eyrie", "model_catalog.json")
}

// CacheInfo reports on-disk cache metadata when present.
func CacheInfo(cachePath string) (exists bool, modTime time.Time, size int64, err error) {
	if cachePath == "" {
		return false, time.Time{}, 0, nil
	}
	info, err := os.Stat(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, time.Time{}, 0, nil
		}
		return false, time.Time{}, 0, err
	}
	return true, info.ModTime(), info.Size(), nil
}

// RefreshCatalogV1 fetches the published catalog, validates it, and writes the cache.
// Unlike LoadCatalogV1 with RefreshRemote, this fails when the remote fetch fails so
// callers never treat a stale cache as a successful refresh.
func RefreshCatalogV1(ctx context.Context, opts LoadCatalogV1Options) (*RefreshResult, error) {
	if opts.CachePath == "" {
		opts.CachePath = DefaultCachePath()
	}
	opts.RemoteURL = ResolvedRemoteCatalogURL(opts.RemoteURL)
	remote, err := FetchRemoteCatalogV1(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("catalog refresh: %w", err)
	}
	if err := WriteCatalogV1Cache(opts.CachePath, remote); err != nil {
		return nil, fmt.Errorf("catalog refresh: write cache: %w", err)
	}
	compiled, err := CompileCatalogV1(remote)
	if err != nil {
		return nil, fmt.Errorf("catalog refresh: compile: %w", err)
	}
	source := "remote"
	if remote.Provenance != nil && remote.Provenance.Source != "" {
		source = remote.Provenance.Source
	}
	return &RefreshResult{
		Compiled:        compiled,
		CachePath:       opts.CachePath,
		Source:          source,
		RemoteURL:       opts.RemoteURL,
		Refreshed:       true,
		RemoteRefreshed: true,
		StaleAfter:      remote.StaleAfter,
	}, nil
}

// Summary returns a one-line human summary for CLI output.
func (r *RefreshResult) Summary() string {
	if r == nil || r.Compiled == nil {
		return "catalog refresh: no data"
	}
	return fmt.Sprintf(
		"Model catalog refreshed (%s): %d models, %d deployments, %d offerings → %s",
		r.Source,
		len(r.Compiled.ModelsByID),
		len(r.Compiled.DeploymentsByID),
		len(r.Compiled.OfferingsByID),
		r.CachePath,
	)
}

// DiscoverReport returns a multi-line report for `hawk models refresh` / `eyrie catalog discover`.
func (r *RefreshResult) DiscoverReport() string {
	if r == nil || r.Compiled == nil {
		return "Catalog discovery: no data"
	}
	var b strings.Builder
	b.WriteString(r.Summary())
	b.WriteString("\n")
	if r.RemoteRefreshed {
		b.WriteString("  Remote catalog: refreshed")
		if r.RemoteURL != "" {
			b.WriteString(" (")
			b.WriteString(r.RemoteURL)
			b.WriteString(")")
		}
		b.WriteString("\n")
	} else {
		b.WriteString("  Remote catalog: using cache or embedded\n")
	}
	if len(r.LiveProviders) == 0 {
		b.WriteString("  Live APIs: none (set API keys in env, e.g. OPENROUTER_API_KEY)\n")
	} else {
		b.WriteString("  Live APIs:\n")
		for _, p := range r.LiveProviders {
			switch {
			case p.Error != "" && strings.HasPrefix(p.Error, "skipped"):
				b.WriteString(fmt.Sprintf("    - %s: %s\n", p.Provider, p.Error))
			case p.Error != "":
				b.WriteString(fmt.Sprintf("    - %s: failed (%s)\n", p.Provider, p.Error))
			default:
				b.WriteString(fmt.Sprintf("    - %s: %d models merged\n", p.Provider, p.ModelCount))
			}
		}
	}
	if !r.StaleAfter.IsZero() {
		b.WriteString(fmt.Sprintf("  stale_after: %s\n", r.StaleAfter.UTC().Format(time.RFC3339)))
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}
