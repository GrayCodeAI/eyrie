package credential

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	learnedPrefixesFile   = "learned_credential_prefixes.json"
	learnedPrefixMinLen   = 4
	learnedPrefixMaxLen   = 12
	learnedMaxPerProvider = 24
)

type learnedPrefixFile struct {
	Version   int                       `json:"version"`
	Providers map[string]map[string]int `json:"providers"`
}

var (
	learnedMu     sync.Mutex
	learnedCached *learnedPrefixFile
)

// RecordLearnedCredential increments local prefix stats after a successful key save.
// Only a short leading prefix is stored — never the full secret.
func RecordLearnedCredential(providerID, secret string) {
	providerID = strings.TrimSpace(providerID)
	prefix := extractLearningPrefix(secret)
	if providerID == "" || prefix == "" {
		return
	}
	learnedMu.Lock()
	defer learnedMu.Unlock()
	data := loadLearnedLocked()
	if data.Providers == nil {
		data.Providers = map[string]map[string]int{}
	}
	counts := data.Providers[providerID]
	if counts == nil {
		counts = map[string]int{}
		data.Providers[providerID] = counts
	}
	counts[prefix]++
	pruneLearnedCounts(counts)
	_ = saveLearnedLocked(data)
	learnedCached = data
}

func learnedPrefixBoost(secret string) map[string]int {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil
	}
	learnedMu.Lock()
	data := loadLearnedLocked()
	learnedMu.Unlock()
	if data == nil || len(data.Providers) == 0 {
		return nil
	}
	out := map[string]int{}
	for providerID, counts := range data.Providers {
		for learned, n := range counts {
			if n <= 0 || learned == "" {
				continue
			}
			if !strings.HasPrefix(secret, learned) {
				continue
			}
			boost := n * 10
			if boost > 120 {
				boost = 120
			}
			if out[providerID] < boost {
				out[providerID] = boost
			}
		}
	}
	return out
}

// extractLearningPrefix returns a short, non-secret fingerprint from the start of a key.
func extractLearningPrefix(secret string) string {
	secret = strings.TrimSpace(secret)
	if len(secret) < learnedPrefixMinLen {
		return ""
	}
	max := learnedPrefixMaxLen
	if len(secret) < max {
		max = len(secret)
	}
	var b strings.Builder
	for i := 0; i < max; i++ {
		r := rune(secret[i])
		if r == '-' || r == '_' || r == '.' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		break
	}
	out := b.String()
	if len(out) < learnedPrefixMinLen {
		return ""
	}
	return out
}

func pruneLearnedCounts(counts map[string]int) {
	if len(counts) <= learnedMaxPerProvider {
		return
	}
	type kv struct {
		k string
		v int
	}
	var all []kv
	for k, v := range counts {
		all = append(all, kv{k, v})
	}
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].v < all[i].v {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	keep := map[string]struct{}{}
	for i := 0; i < learnedMaxPerProvider && i < len(all); i++ {
		keep[all[i].k] = struct{}{}
	}
	for k := range counts {
		if _, ok := keep[k]; !ok {
			delete(counts, k)
		}
	}
}

func learnedHawkConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".hawk"
	}
	return filepath.Join(home, ".hawk")
}

func learnedPrefixesPath() string {
	return filepath.Join(learnedHawkConfigDir(), learnedPrefixesFile)
}

func loadLearnedLocked() *learnedPrefixFile {
	if learnedCached != nil {
		return learnedCached
	}
	path := learnedPrefixesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		learnedCached = &learnedPrefixFile{Version: 1, Providers: map[string]map[string]int{}}
		return learnedCached
	}
	var f learnedPrefixFile
	if json.Unmarshal(data, &f) != nil || f.Providers == nil {
		learnedCached = &learnedPrefixFile{Version: 1, Providers: map[string]map[string]int{}}
		return learnedCached
	}
	if f.Version == 0 {
		f.Version = 1
	}
	learnedCached = &f
	return learnedCached
}

func saveLearnedLocked(f *learnedPrefixFile) error {
	if f == nil {
		return nil
	}
	if f.Version == 0 {
		f.Version = 1
	}
	dir := learnedHawkConfigDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(learnedPrefixesPath(), raw, 0o600)
}

// InvalidateLearnedPrefixCache clears the in-memory learned-prefix snapshot (tests).
func InvalidateLearnedPrefixCache() {
	learnedMu.Lock()
	learnedCached = nil
	learnedMu.Unlock()
}

// isGenericOpenAIShapedKey reports OpenAI-style keys without a known vendor prefix.
func isGenericOpenAIShapedKey(secret string) bool {
	secret = strings.TrimSpace(secret)
	if !strings.HasPrefix(secret, "sk-") {
		return false
	}
	if strings.HasPrefix(secret, "sk-ant-") ||
		strings.HasPrefix(secret, "sk-or-") ||
		strings.HasPrefix(secret, "sk-proj-") ||
		strings.HasPrefix(secret, "sk-svcacct-") {
		return false
	}
	return true
}