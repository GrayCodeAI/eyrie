package credential

import "context"

type ctxProbeDisambigKey struct{}

// ContextWithoutProbeDisambiguation skips live API probes during ResolveCredential (tests).
func ContextWithoutProbeDisambiguation(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxProbeDisambigKey{}, true)
}
