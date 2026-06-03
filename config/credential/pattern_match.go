package credential

import (
	"sort"
	"strings"

	"github.com/GrayCodeAI/eyrie/catalog/registry"
)

// extraContainsRules are substring hints not stored on ProviderSpec (de facto / scanner-derived).
var extraContainsRules = map[string][]string{
	"openai": {"T3BlbkFJ"},
}

type providerMatch struct {
	providerID string
	score      int
	inferred   bool
}

func rankedProviderMatches(secret string) []providerMatch {
	secret = strings.TrimSpace(secret)
	learned := learnedPrefixBoost(secret)
	var matches []providerMatch
	seen := map[string]bool{}

	add := func(id string, score int, inferred bool) {
		if id == "" || score <= 0 {
			return
		}
		if prev, ok := seen[id]; ok {
			for i := range matches {
				if matches[i].providerID != id {
					continue
				}
				if score > matches[i].score {
					matches[i].score = score
				}
				if inferred {
					matches[i].inferred = true
				}
				_ = prev
				return
			}
			return
		}
		seen[id] = true
		matches = append(matches, providerMatch{providerID: id, score: score, inferred: inferred})
	}

	for _, spec := range registry.DefaultRegistry.CredentialProviders() {
		if !spec.RequiresKey {
			continue
		}
		best := 0
		inferred := false
		for pi, prefix := range spec.KeyPrefixes {
			if prefix == "" || !strings.HasPrefix(secret, prefix) {
				continue
			}
			score := 1000 + len(prefix)*10 + (100 - spec.SortOrder) - pi
			if score > best {
				best = score
			}
			inferred = true
		}
		for _, sub := range extraContainsRules[spec.ProviderID] {
			if sub != "" && strings.Contains(secret, sub) {
				score := 850 + (100 - spec.SortOrder)
				if score > best {
					best = score
				}
				inferred = true
			}
		}
		if boost := learned[spec.ProviderID]; boost > 0 {
			score := 500 + boost
			if score > best {
				best = score
			}
			if boost >= 20 {
				inferred = true
			}
		}
		if best > 0 {
			add(spec.ProviderID, best, inferred)
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].providerID < matches[j].providerID
	})
	return matches
}

func inferredSetFromMatches(matches []providerMatch) map[string]int {
	out := map[string]int{}
	for rank, m := range matches {
		if !m.inferred {
			continue
		}
		out[m.providerID] = rank
	}
	return out
}