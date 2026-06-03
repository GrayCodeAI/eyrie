package credentials

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// MigrateLegacyEnvFile imports API keys from legacy plaintext credential files
// (~/.hawk/env, ~/.hawk/.env) into the OS secret store and removes them.
// It also copies legacy keychain account names (e.g. xiaomi_mimo_api_key → payg).
func legacyEnvMigrationMarkerPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".hawk", ".legacy-env-migrated")
}

func legacyEnvMigrationDone() bool {
	_, err := os.Stat(legacyEnvMigrationMarkerPath())
	return err == nil
}

func markLegacyEnvMigrationDone() {
	_ = os.MkdirAll(filepath.Dir(legacyEnvMigrationMarkerPath()), 0o700)
	_ = os.WriteFile(legacyEnvMigrationMarkerPath(), []byte("ok\n"), 0o600)
}

func MigrateLegacyEnvFile(ctx context.Context) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if legacyEnvMigrationDone() {
		n, _ := MigrateLegacyKeychainAccounts(ctx)
		return n, nil
	}
	total := 0
	for _, path := range []string{legacyEnvPath(), legacyHawkDotEnvPath()} {
		n, err := migrateLegacyEnvFileAt(ctx, path)
		if err != nil && !os.IsNotExist(err) {
			return total, err
		}
		total += n
	}
	n, err := MigrateLegacyKeychainAccounts(ctx)
	if err != nil {
		return total, err
	}
	total += n
	markLegacyEnvMigrationDone()
	return total, nil
}

var legacyKeychainAccountCopies = []struct{ from, to string }{
	{"xiaomi_mimo_api_key", "xiaomi_mimo_payg_api_key"},
}

// MigrateLegacyKeychainAccounts copies secrets from deprecated keychain accounts when the new account is empty.
func MigrateLegacyKeychainAccounts(ctx context.Context) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cs, ok := DefaultStore().(*CombinedStore)
	if !ok || cs.Keychain == nil {
		return 0, nil
	}
	migrated := 0
	for _, pair := range legacyKeychainAccountCopies {
		existing, err := cs.Keychain.Get(ctx, pair.to)
		if err == nil && strings.TrimSpace(existing) != "" {
			continue
		}
		legacy, err := cs.Keychain.Get(ctx, pair.from)
		if err != nil || strings.TrimSpace(legacy) == "" {
			continue
		}
		if err := cs.Keychain.Set(ctx, pair.to, strings.TrimSpace(legacy)); err != nil {
			continue
		}
		migrated++
	}
	return migrated, nil
}

func migrateLegacyEnvFileAt(ctx context.Context, path string) (int, error) {
	secrets, err := readLegacyEnvFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if len(secrets) == 0 {
		_ = os.Remove(path)
		return 0, nil
	}
	cs, ok := DefaultStore().(*CombinedStore)
	if !ok || cs.Keychain == nil {
		return 0, ErrKeychainUnavailable
	}
	migrated := 0
	for _, envKey := range discoveryEnvKeys(ctx) {
		secret := strings.TrimSpace(secrets[envKey])
		if secret == "" {
			continue
		}
		account := AccountForEnv(envKey)
		if existing, err := cs.Keychain.Get(ctx, account); err == nil && strings.TrimSpace(existing) != "" {
			migrated++
			continue
		}
		if err := cs.Keychain.Set(ctx, account, secret); err != nil {
			continue
		}
		migrated++
	}
	if migrated > 0 || len(secrets) > 0 {
		_ = os.Remove(path)
	}
	return migrated, nil
}

func legacyEnvPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".hawk", "env")
}

func legacyHawkDotEnvPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".hawk", ".env")
}

func readLegacyEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// Strip surrounding quotes (single or double) from values.
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if key != "" && val != "" {
			out[key] = val
		}
	}
	return out, nil
}
