package adapters

import (
	"os"
	"testing"
)

func TestDynamicProviderEnabled_EnvNotSet(t *testing.T) {
	os.Unsetenv(DynamicProviderEnvVar)
	if DynamicProviderEnabled() {
		t.Error("expected false when env var is not set")
	}
}

func TestDynamicProviderEnabled_EnvSetTo1(t *testing.T) {
	os.Setenv(DynamicProviderEnvVar, "1")
	defer os.Unsetenv(DynamicProviderEnvVar)
	if !DynamicProviderEnabled() {
		t.Error("expected true when env var is '1'")
	}
}

func TestDynamicProviderEnabled_EnvSetToTrue(t *testing.T) {
	os.Setenv(DynamicProviderEnvVar, "true")
	defer os.Unsetenv(DynamicProviderEnvVar)
	if !DynamicProviderEnabled() {
		t.Error("expected true when env var is 'true'")
	}
}

func TestDynamicProviderEnabled_EnvSetToYes(t *testing.T) {
	os.Setenv(DynamicProviderEnvVar, "yes")
	defer os.Unsetenv(DynamicProviderEnvVar)
	if !DynamicProviderEnabled() {
		t.Error("expected true when env var is 'yes'")
	}
}

func TestDynamicProviderEnabled_EnvSetToNo(t *testing.T) {
	os.Setenv(DynamicProviderEnvVar, "no")
	defer os.Unsetenv(DynamicProviderEnvVar)
	if DynamicProviderEnabled() {
		t.Error("expected false when env var is 'no'")
	}
}

func TestFreezeRegistry(t *testing.T) {
	FreezeRegistry()
	if !registryFrozen.Load() {
		t.Error("expected registry to be frozen after FreezeRegistry")
	}
	// Reset for other tests
	registryFrozen.Store(false)
}

func TestRegisterDynamicProvider_Success(t *testing.T) {
	registryFrozen.Store(false)
	// Save and restore the map
	saved := OpenAICompatibleProviders
	OpenAICompatibleProviders = make(map[string]ProviderRegistryConfig)
	defer func() { OpenAICompatibleProviders = saved }()

	err := RegisterDynamicProvider("my-provider", "https://my-api.example.com", "MY_API_KEY")
	if err != nil {
		t.Fatalf("RegisterDynamicProvider failed: %v", err)
	}
	p, ok := OpenAICompatibleProviders["my-provider"]
	if !ok {
		t.Fatal("expected my-provider to be registered")
	}
	if p.Type != ProviderTypeOpenAICompatible {
		t.Errorf("expected type openai-compatible, got %s", p.Type)
	}
	if p.BaseURL != "https://my-api.example.com" {
		t.Errorf("expected base URL https://my-api.example.com, got %s", p.BaseURL)
	}
	if p.EnvKey != "MY_API_KEY" {
		t.Errorf("expected env key MY_API_KEY, got %s", p.EnvKey)
	}
}

func TestRegisterDynamicProvider_Frozen(t *testing.T) {
	registryFrozen.Store(true)
	defer registryFrozen.Store(false)

	err := RegisterDynamicProvider("test", "https://example.com", "KEY")
	if err == nil {
		t.Fatal("expected error when registry is frozen")
	}
}

func TestRegisterDynamicProvider_EmptyBaseURL(t *testing.T) {
	registryFrozen.Store(false)
	err := RegisterDynamicProvider("test", "", "KEY")
	if err == nil {
		t.Fatal("expected error for empty baseURL")
	}
}

func TestRegisterDynamicProvider_InvalidURL(t *testing.T) {
	registryFrozen.Store(false)
	err := RegisterDynamicProvider("test", "not-a-url", "KEY")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestRegisterDynamicProvider_NoScheme(t *testing.T) {
	registryFrozen.Store(false)
	err := RegisterDynamicProvider("test", "example.com/api", "KEY")
	if err == nil {
		t.Fatal("expected error for URL without scheme")
	}
}

func TestOpenAIBaseFallbackURL_APIBASE(t *testing.T) {
	os.Setenv("OPENAI_API_BASE", "https://api.example.com/v1")
	defer os.Unsetenv("OPENAI_API_BASE")
	os.Unsetenv("OPENAI_BASE_URL")

	u := OpenAIBaseFallbackURL()
	if u != "https://api.example.com/v1" {
		t.Errorf("expected OPENAI_API_BASE value, got %q", u)
	}
}

func TestOpenAIBaseFallbackURL_BASEURL(t *testing.T) {
	os.Unsetenv("OPENAI_API_BASE")
	os.Setenv("OPENAI_BASE_URL", "https://alt.example.com")
	defer os.Unsetenv("OPENAI_BASE_URL")

	u := OpenAIBaseFallbackURL()
	if u != "https://alt.example.com" {
		t.Errorf("expected OPENAI_BASE_URL value, got %q", u)
	}
}

func TestOpenAIBaseFallbackURL_PrefersAPIBASE(t *testing.T) {
	os.Setenv("OPENAI_API_BASE", "https://api.example.com")
	defer os.Unsetenv("OPENAI_API_BASE")
	os.Setenv("OPENAI_BASE_URL", "https://alt.example.com")
	defer os.Unsetenv("OPENAI_BASE_URL")

	u := OpenAIBaseFallbackURL()
	if u != "https://api.example.com" {
		t.Errorf("expected OPENAI_API_BASE to take priority, got %q", u)
	}
}

func TestOpenAIBaseFallbackURL_NotSet(t *testing.T) {
	os.Unsetenv("OPENAI_API_BASE")
	os.Unsetenv("OPENAI_BASE_URL")

	u := OpenAIBaseFallbackURL()
	if u != "" {
		t.Errorf("expected empty string, got %q", u)
	}
}
