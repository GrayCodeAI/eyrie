package engine

import (
	"path/filepath"
	"sync"

	"github.com/GrayCodeAI/eyrie/config"
)

var providerStateLocks sync.Map

func lockProviderStatePath(path string) func() {
	key, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		key = filepath.Clean(path)
	}
	value, _ := providerStateLocks.LoadOrStore(key, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func (e *Engine) loadProviderConfigStrict() (*config.ProviderConfig, error) {
	cfg, err := config.LoadProviderConfigWithError(e.providerConfigPath)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = &config.ProviderConfig{}
	}
	return cfg, nil
}
