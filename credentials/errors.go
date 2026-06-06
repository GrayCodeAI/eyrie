package credentials

import "errors"

var ErrNotFound = errors.New("credentials: not found")

// ErrNoOIDC is returned by the OIDC convenience helpers when the process is not
// running inside a GitHub Actions runner, so callers can fall back to other
// credential sources.
var ErrNoOIDC = errors.New("credentials: not running in GitHub Actions (no OIDC)")
