package api

import "testing"

func TestConstantTimeEqual(t *testing.T) {
	key := "super-secret-api-key"
	if !constantTimeEqual(key, key) {
		t.Fatal("expected equal tokens to match")
	}
	if constantTimeEqual(key, key+"x") {
		t.Fatal("expected different-length tokens to not match")
	}
	if constantTimeEqual("short", "much-longer-token") {
		t.Fatal("expected mismatched tokens to not match")
	}
}
