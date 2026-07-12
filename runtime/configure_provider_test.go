package runtime

import (
	"context"
	"strings"
	"testing"
)

func TestConfigureProviderRejectsUnknownProvider(t *testing.T) {
	_, err := ConfigureProvider(context.Background(), ConfigureProviderOpts{ProviderID: "missing-provider", Secret: "secret-value"})
	if err == nil || !strings.Contains(err.Error(), "unknown setup provider") {
		t.Fatalf("ConfigureProvider() error = %v", err)
	}
}

func TestConfigureProviderRejectsEmptyValueBeforeNetwork(t *testing.T) {
	_, err := ConfigureProvider(context.Background(), ConfigureProviderOpts{ProviderID: "openai"})
	if err == nil || !strings.Contains(err.Error(), "credential value required") {
		t.Fatalf("ConfigureProvider() error = %v", err)
	}
}
