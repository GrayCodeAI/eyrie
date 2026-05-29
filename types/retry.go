package types

import "time"

// RetryConfig is the canonical retry configuration shared by HTTP clients and
// the router. Packages that need extra fields (e.g. RetryOn for status codes,
// OnRetry for callbacks) embed this type and add the extensions.
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}
