package credential

import (
	"context"
	"sort"
	"sync"

	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

type ctxProbeDisambigKey struct{}

// ContextWithoutProbeDisambiguation skips live API probes during ResolveCredential (tests).
func ContextWithoutProbeDisambiguation(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxProbeDisambigKey{}, true)
}

func probeDisambiguationEnabled(ctx context.Context) bool {
	// Keys setup is provider-first; no multi-gateway probe prefetch on paste.
	_ = ctx
	return false
}

const maxProbeDisambiguation = 6

// applyProbeDisambiguation verifies ambiguous keys against provider HTTP APIs in parallel.
func applyProbeDisambiguation(ctx context.Context, secret string, options []CredentialProviderOption) ([]CredentialProviderOption, bool) {
	if !probeDisambiguationEnabled(ctx) {
		return options, false
	}
	matches := rankedProviderMatches(secret)
	if len(inferredSetFromMatches(matches)) > 0 {
		return options, false
	}
	if !isGenericOpenAIShapedKey(secret) {
		return options, false
	}

	var candidates []CredentialProviderOption
	for _, opt := range options {
		spec, ok := registry.DefaultRegistry.Get(opt.ProviderID)
		if !ok || !spec.RequiresKey || spec.ProbeKind == "" || spec.ProbeKind == registry.ProbeNone {
			continue
		}
		candidates = append(candidates, opt)
		if len(candidates) >= maxProbeDisambiguation {
			break
		}
	}
	if len(candidates) == 0 {
		return options, false
	}

	verified := probeCandidatesParallel(ctx, secret, candidates)
	if len(verified) == 0 {
		return options, false
	}

	out := make([]CredentialProviderOption, len(options))
	copy(out, options)
	for i := range out {
		if verified[out[i].ProviderID] {
			out[i].Inferred = true
			out[i].Rank = out[i].Rank - 2000
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Inferred != out[j].Inferred {
			return out[i].Inferred
		}
		if out[i].Rank != out[j].Rank {
			return out[i].Rank < out[j].Rank
		}
		return out[i].DisplayName < out[j].DisplayName
	})
	return out, true
}

func probeCandidatesParallel(ctx context.Context, secret string, candidates []CredentialProviderOption) map[string]bool {
	type probeResult struct {
		providerID string
		ok         bool
	}
	ch := make(chan probeResult, len(candidates))
	var wg sync.WaitGroup
	for _, opt := range candidates {
		wg.Add(1)
		go func(o CredentialProviderOption) {
			defer wg.Done()
			err := ProbeCredential(ctx, o.EnvVar, secret)
			ch <- probeResult{o.ProviderID, err == nil}
		}(opt)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()
	verified := map[string]bool{}
	for r := range ch {
		if r.ok {
			verified[r.providerID] = true
		}
	}
	return verified
}