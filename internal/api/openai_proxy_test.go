//nolint:bodyclose,noctx
package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestOpenAIChatCompletionsNonStream(t *testing.T) {
	t.Parallel()
	ts := testServer(t)
	defer ts.Close()

	body := `{"model":"gpt-4o","messages":[{"role":"system","content":"be terse"},{"role":"user","content":"hi"}]}`
	resp, err := http.Post(ts.URL+"/v1/chat/completions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var out openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Object != "chat.completion" {
		t.Errorf("object = %q, want chat.completion", out.Object)
	}
	if !strings.HasPrefix(out.ID, "chatcmpl-") {
		t.Errorf("id = %q, want chatcmpl- prefix", out.ID)
	}
	if out.Model != "gpt-4o" {
		t.Errorf("model = %q, want gpt-4o", out.Model)
	}
	if len(out.Choices) != 1 {
		t.Fatalf("choices = %d, want 1", len(out.Choices))
	}
	c := out.Choices[0]
	if c.Message.Role != "assistant" || c.Message.Content != "hi" {
		t.Errorf("choice message = %+v, want assistant/hi", c.Message)
	}
	if c.FinishReason != "stop" {
		t.Errorf("finish_reason = %q, want stop", c.FinishReason)
	}
	// mockProv reports 2 completion tokens; usage should be surfaced.
	if out.Usage.CompletionTokens != 2 {
		t.Errorf("completion_tokens = %d, want 2", out.Usage.CompletionTokens)
	}
	if out.Usage.TotalTokens != out.Usage.PromptTokens+out.Usage.CompletionTokens {
		t.Errorf("total_tokens mismatch: %+v", out.Usage)
	}
}

func TestOpenAIChatCompletionsStream(t *testing.T) {
	t.Parallel()
	ts := testServer(t)
	defer ts.Close()

	body := `{"model":"gpt-4o","stream":true,"messages":[{"role":"user","content":"hi"}]}`
	resp, err := http.Post(ts.URL+"/v1/chat/completions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}

	var (
		dataLines  []string
		sawDone    bool
		sawContent bool
		sawFinish  bool
	)
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			sawDone = true
			continue
		}
		dataLines = append(dataLines, payload)

		var chunk openAIChatChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			t.Fatalf("unmarshal chunk %q: %v", payload, err)
		}
		if chunk.Object != "chat.completion.chunk" {
			t.Errorf("chunk object = %q, want chat.completion.chunk", chunk.Object)
		}
		if len(chunk.Choices) == 1 {
			if chunk.Choices[0].Delta.Content == "hi" {
				sawContent = true
			}
			if chunk.Choices[0].FinishReason != nil {
				sawFinish = true
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(dataLines) == 0 {
		t.Fatal("expected at least one data chunk")
	}
	if !sawContent {
		t.Error("expected a content delta chunk")
	}
	if !sawFinish {
		t.Error("expected a terminal chunk with finish_reason")
	}
	if !sawDone {
		t.Error("expected a [DONE] sentinel")
	}
}

func TestOpenAIChatCompletionsRequiresMessages(t *testing.T) {
	t.Parallel()
	ts := testServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"m","messages":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSplitOpenAIMessages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		messages   []openAIChatMessage
		wantSystem string
		wantPrompt string
	}{
		{
			name:       "single user",
			messages:   []openAIChatMessage{{Role: "user", Content: "hello"}},
			wantSystem: "",
			wantPrompt: "hello",
		},
		{
			name: "system plus user",
			messages: []openAIChatMessage{
				{Role: "system", Content: "be nice"},
				{Role: "user", Content: "hello"},
			},
			wantSystem: "be nice",
			wantPrompt: "hello",
		},
		{
			name: "multi turn folds transcript",
			messages: []openAIChatMessage{
				{Role: "user", Content: "first"},
				{Role: "assistant", Content: "reply"},
				{Role: "user", Content: "second"},
			},
			wantSystem: "",
			wantPrompt: "User: first\n\nAssistant: reply\n\nUser: second",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			system, prompt := splitOpenAIMessages(tt.messages)
			if system != tt.wantSystem {
				t.Errorf("system = %q, want %q", system, tt.wantSystem)
			}
			if prompt != tt.wantPrompt {
				t.Errorf("prompt = %q, want %q", prompt, tt.wantPrompt)
			}
		})
	}
}

func TestOpenAIFinishReason(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"", "stop"},
		{"end_turn", "stop"},
		{"max_tokens", "length"},
		{"tool_use", "tool_calls"},
		{"weird", "stop"},
	}
	for _, tt := range tests {
		if got := openAIFinishReason(tt.in); got != tt.want {
			t.Errorf("openAIFinishReason(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
