package credentials

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// StorageReport summarizes where credentials are stored (no secret values).
type StorageReport struct {
	PlatformStore    string
	KeychainWritable bool
	KeychainDetail   string
	StoredEnvKeys    []string
}

// StorageReportFor returns a credential storage summary for CLI / doctor output.
func StorageReportFor(ctx context.Context) StorageReport {
	if ctx == nil {
		ctx = context.Background()
	}
	stored := StoredEnvKeys(ctx)
	report := StorageReport{
		PlatformStore: PlatformSecretStoreName(),
		StoredEnvKeys: stored,
	}
	ok, detail := KeychainWriteAvailable(ctx)
	report.KeychainWritable = ok
	report.KeychainDetail = detail
	return report
}

// StoredEnvKeys returns env var names that have non-empty secrets in the store.
func StoredEnvKeys(ctx context.Context) []string {
	if ctx == nil {
		ctx = context.Background()
	}
	var keys []string
	for _, envKey := range discoveryEnvKeys(ctx) {
		if HasSecret(ctx, envKey) {
			keys = append(keys, envKey)
		}
	}
	sort.Strings(keys)
	return keys
}

// FormatStorageReport returns human-readable credential storage status.
func FormatStorageReport(r StorageReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Credential storage: %s only\n", r.PlatformStore)
	if r.KeychainWritable {
		fmt.Fprintf(&b, "  Keychain: writable\n")
	} else {
		fmt.Fprintf(&b, "  Keychain: %s\n", r.KeychainDetail)
	}
	if len(r.StoredEnvKeys) == 0 {
		fmt.Fprintf(&b, "  Stored: (none)\n")
	} else {
		fmt.Fprintf(&b, "  Stored:\n")
		for _, key := range r.StoredEnvKeys {
			fmt.Fprintf(&b, "    %s\n", key)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
