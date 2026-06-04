package credential

import "context"

type ctxProbeDisambigKey struct{}

// ContextWithoutProbeDisambiguation skips live API probes during ResolveCredential (tests).
func ContextWithoutProbeDisambiguation(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxProbeDisambigKey{}, true)
}

// applyProbeDisambiguation is disabled: setup selects gateway before pasting keys.
func applyProbeDisambiguation(ctx context.Context, secret string, options []CredentialProviderOption) ([]CredentialProviderOption, bool) {
	_ = ctx
	_ = secret
	return options, false
}
