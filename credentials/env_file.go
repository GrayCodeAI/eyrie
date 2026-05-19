package credentials

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
)

// EnvFileStore reads ~/.hawk/env (hawk convention) for fallback and CI.
type EnvFileStore struct{}

func (e *EnvFileStore) Set(ctx context.Context, account, secret string) error {
	_ = ctx
	path := hawkEnvPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	lines := readEnvLines(path)
	envKey := EnvForAccount(account)
	var kept []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "export "+envKey+"=") {
			kept = append(kept, line)
		}
	}
	kept = append(kept, "export "+envKey+"="+secret)
	return os.WriteFile(path, []byte(strings.Join(kept, "\n")+"\n"), 0o600)
}

func (e *EnvFileStore) Get(ctx context.Context, account string) (string, error) {
	_ = ctx
	envKey := EnvForAccount(account)
	for _, line := range readEnvLines(hawkEnvPath()) {
		if !strings.HasPrefix(line, "export ") {
			continue
		}
		rest := strings.TrimPrefix(line, "export ")
		idx := strings.Index(rest, "=")
		if idx < 0 {
			continue
		}
		if strings.TrimSpace(rest[:idx]) != envKey {
			continue
		}
		v := strings.TrimSpace(rest[idx+1:])
		if v != "" {
			return v, nil
		}
	}
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v, nil
	}
	return "", ErrNotFound
}

func (e *EnvFileStore) Delete(ctx context.Context, account string) error {
	_ = ctx
	path := hawkEnvPath()
	lines := readEnvLines(path)
	envKey := EnvForAccount(account)
	var kept []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "export "+envKey+"=") {
			kept = append(kept, line)
		}
	}
	if len(kept) == 0 {
		return os.Remove(path)
	}
	return os.WriteFile(path, []byte(strings.Join(kept, "\n")+"\n"), 0o600)
}

func hawkEnvPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".hawk", "env")
}

func readEnvLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// LoadEnvFileIntoProcess applies ~/.hawk/env without overriding existing env.
func LoadEnvFileIntoProcess() error {
	path := hawkEnvPath()
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.HasPrefix(line, "export ") {
			continue
		}
		rest := strings.TrimPrefix(line, "export ")
		idx := strings.Index(rest, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(rest[:idx])
		val := strings.TrimSpace(rest[idx+1:])
		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
	return sc.Err()
}
