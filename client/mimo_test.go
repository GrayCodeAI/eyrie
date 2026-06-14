package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/GrayCodeAI/eyrie/catalog/xiaomi"
	eyriecfg "github.com/GrayCodeAI/eyrie/config"
)

func TestMimoRetryableChatError_HTTPStatus(t *testing.T) {
	err401 := errors.New("eyrie: openai API error: credential probe failed: invalid API key (HTTP 401)")
	if !mimoRetryableChatError(err401) {
		t.Fatal("expected 401 retryable")
	}
	err400 := errors.New("eyrie: openai API error (HTTP 400)")
	if mimoRetryableChatError(err400) {
		t.Fatal("expected 400 not retryable")
	}
}

func TestMimoRetryableChatError_UsesXiaomiMimoHelper(t *testing.T) {
	if !xiaomi.IsRetryableHTTPStatus(http.StatusServiceUnavailable) {
		t.Fatal("expected 503 retryable in xiaomi helper")
	}
	err := errors.New("provider unavailable (HTTP 503)")
	if !mimoRetryableChatError(err) {
		t.Fatal("expected 503 retryable")
	}
}

func TestMimoFallbackChatError_ParamIncorrect(t *testing.T) {
	err := errors.New("eyrie: xiaomi_mimo_token_plan API error (request_id=): : Param Incorrect")
	if !mimoFallbackChatError(err) {
		t.Fatal("expected Param Incorrect to fallback to Anthropic compatibility")
	}
	if mimoFallbackChatError(errors.New("eyrie: openai API error: invalid model")) {
		t.Fatal("non-MiMo unrelated errors should not trigger fallback")
	}
}

func TestMiMoClient_ChatFallsBackToAnthropicOnParamIncorrect(t *testing.T) {
	anthropicCalls := 0
	openAITransport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("openai path = %q", r.URL.Path)
		}
		return jsonResponse(http.StatusBadRequest, map[string]interface{}{"error": map[string]string{"message": "Param Incorrect"}}), nil
	})
	anthropicTransport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		anthropicCalls++
		if r.URL.Path != "/anthropic/v1/messages" {
			t.Fatalf("anthropic path = %q", r.URL.Path)
		}
		if r.Header.Get("api-key") != "tp-test-key" {
			t.Fatalf("missing MiMo api-key auth header")
		}
		return jsonResponse(http.StatusOK, map[string]interface{}{
			"id":          "msg_1",
			"type":        "message",
			"role":        "assistant",
			"content":     []map[string]string{{"type": "text", "text": "ok"}},
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 1, "output_tokens": 1},
		}), nil
	})

	c := NewMiMoClient("tp-test-key", "https://openai.example/v1", "https://anthropic.example/anthropic", &XiaomiMimoCompat, "xiaomi_mimo_token_plan")
	c.router.OpenAI.httpClient = &http.Client{Transport: openAITransport}
	c.router.Anthropic.httpClient = &http.Client{Transport: anthropicTransport}
	resp, err := c.Chat(context.Background(), []EyrieMessage{{Role: "user", Content: "hi"}}, ChatOptions{
		Model:     "mimo-v2.5-pro",
		MaxTokens: 1024,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("content = %q, want ok", resp.Content)
	}
	if anthropicCalls != 1 {
		t.Fatalf("anthropic calls = %d, want 1", anthropicCalls)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(status int, payload interface{}) *http.Response {
	var body bytes.Buffer
	_ = json.NewEncoder(&body).Encode(payload)
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(body.Bytes())),
	}
}

func TestGetOrCreateProvider_XiaomiMimoTokenPlanUsesMimoBase(t *testing.T) {
	t.Setenv("HAWK_CONFIG_DIR", t.TempDir())
	if err := eyriecfg.SaveProviderConfig(&eyriecfg.ProviderConfig{
		XiaomiMimoTokenPlanRegion: "sgp",
	}, ""); err != nil {
		t.Fatalf("SaveProviderConfig: %v", err)
	}

	c := Client(&EyrieConfig{Provider: "xiaomi_mimo_token_plan", APIKey: "tp-test-key"})
	p, err := c.getOrCreateProvider("xiaomi_mimo_token_plan")
	if err != nil {
		t.Fatalf("getOrCreateProvider: %v", err)
	}
	mimo, ok := p.(*MiMoClient)
	if !ok {
		t.Fatalf("provider type = %T, want *MiMoClient", p)
	}
	if mimo.router.OpenAI.baseURL != xiaomi.TokenPlanSGPOpenAIBase {
		t.Fatalf("openAI baseURL = %q, want %q", mimo.router.OpenAI.baseURL, xiaomi.TokenPlanSGPOpenAIBase)
	}
	if mimo.router.Anthropic == nil || mimo.router.Anthropic.baseURL != xiaomi.TokenPlanSGPAnthropicBase {
		t.Fatalf("anthropic baseURL = %#v, want %q", mimo.router.Anthropic, xiaomi.TokenPlanSGPAnthropicBase)
	}
}
