//nolint:bodyclose,noctx
package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GrayCodeAI/eyrie/client"
	"github.com/GrayCodeAI/eyrie/storage"
)

// --- Configurable mock providers ---

// errorProvider returns an error from Chat/StreamChat.
type errorProvider struct {
	err error
}

func (e *errorProvider) Name() string                 { return "error-provider" }
func (e *errorProvider) Ping(_ context.Context) error { return nil }
func (e *errorProvider) Chat(_ context.Context, _ []client.EyrieMessage, _ client.ChatOptions) (*client.EyrieResponse, error) {
	return nil, e.err
}

func (e *errorProvider) StreamChat(_ context.Context, _ []client.EyrieMessage, _ client.ChatOptions) (*client.StreamResult, error) {
	return nil, e.err
}

// slowProvider simulates a provider that takes time to respond.
//
//nolint:unused
type slowProvider struct { //nolint:unused
	delay time.Duration
}

func (s *slowProvider) Name() string                 { return "slow-provider" } //nolint:unused
func (s *slowProvider) Ping(_ context.Context) error { return nil }             //nolint:unused

//nolint:unused
func (s *slowProvider) Chat(_ context.Context, _ []client.EyrieMessage, _ client.ChatOptions) (*client.EyrieResponse, error) {
	time.Sleep(s.delay)
	return &client.EyrieResponse{Content: "slow response", FinishReason: "end_turn", Usage: &client.EyrieUsage{CompletionTokens: 2}}, nil
}

//nolint:unused
func (s *slowProvider) StreamChat(_ context.Context, _ []client.EyrieMessage, _ client.ChatOptions) (*client.StreamResult, error) {
	ch := make(chan client.EyrieStreamEvent, 2)
	go func() {
		time.Sleep(s.delay)
		ch <- client.EyrieStreamEvent{Type: "content", Content: "slow stream"}
		ch <- client.EyrieStreamEvent{Type: "done", StopReason: "end_turn", Usage: &client.EyrieUsage{CompletionTokens: 2}}
		close(ch)
	}()
	return &client.StreamResult{Events: ch}, nil
}

// streamingProvider returns multiple content chunks.
type streamingProvider struct{}

func (s *streamingProvider) Name() string                 { return "streaming-provider" }
func (s *streamingProvider) Ping(_ context.Context) error { return nil }
func (s *streamingProvider) Chat(_ context.Context, _ []client.EyrieMessage, _ client.ChatOptions) (*client.EyrieResponse, error) {
	return &client.EyrieResponse{Content: "hello world", FinishReason: "end_turn", Usage: &client.EyrieUsage{CompletionTokens: 2}}, nil
}

func (s *streamingProvider) StreamChat(_ context.Context, _ []client.EyrieMessage, _ client.ChatOptions) (*client.StreamResult, error) {
	ch := make(chan client.EyrieStreamEvent, 5)
	go func() {
		chunks := []string{"hello", " ", "world"}
		for _, c := range chunks {
			ch <- client.EyrieStreamEvent{Type: "content", Content: c}
		}
		ch <- client.EyrieStreamEvent{Type: "done", StopReason: "end_turn", Usage: &client.EyrieUsage{CompletionTokens: 3}}
		close(ch)
	}()
	return &client.StreamResult{Events: ch}, nil
}

// errorStreamProvider streams a content chunk then emits an error.
type errorStreamProvider struct{}

func (e *errorStreamProvider) Name() string                 { return "error-stream" }
func (e *errorStreamProvider) Ping(_ context.Context) error { return nil }
func (e *errorStreamProvider) Chat(_ context.Context, _ []client.EyrieMessage, _ client.ChatOptions) (*client.EyrieResponse, error) {
	return nil, fmt.Errorf("provider error")
}

func (e *errorStreamProvider) StreamChat(_ context.Context, _ []client.EyrieMessage, _ client.ChatOptions) (*client.StreamResult, error) {
	ch := make(chan client.EyrieStreamEvent, 3)
	go func() {
		ch <- client.EyrieStreamEvent{Type: "content", Content: "partial"}
		ch <- client.EyrieStreamEvent{Type: "error", Error: "rate limit exceeded"}
		close(ch)
	}()
	return &client.StreamResult{Events: ch}, nil
}

// --- Helper functions ---

func testServerWithProvider(t *testing.T, prov client.Provider) *httptest.Server {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	srv := NewServer(Config{Store: store, Provider: prov})
	return httptest.NewServer(srv)
}

func testServerWithAPIKey(t *testing.T, prov client.Provider, apiKey string) *httptest.Server {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	srv := NewServer(Config{Store: store, Provider: prov, APIKey: apiKey})
	return httptest.NewServer(srv)
}

func jsonBody(msg string) io.Reader {
	return strings.NewReader(msg)
}

func parseJSON(t *testing.T, body io.Reader) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

// --- Health Endpoint Tests ---

func TestHealthEndpoint_ReturnsOK(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body := parseJSON(t, resp.Body)
	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", body["status"])
	}
}

func TestHealthEndpoint_ReturnsJSON(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json content-type, got %s", ct)
	}
}

func TestHealthEndpoint_NoAuthRequired(t *testing.T) {
	// Health endpoint should work without authentication even when API key is set.
	ts := testServerWithAPIKey(t, &mockProv{}, "secret")
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health should not require auth, got %d", resp.StatusCode)
	}
}

// --- Chat Endpoint Tests (non-streaming) ---

func TestPrompt_ReturnsContentAndNodeID(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/prompt", "application/json", jsonBody(`{"message":"hello","model":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	result := parseJSON(t, resp.Body)
	_ = resp.Body.Close()

	if result["content"] != "hi" {
		t.Errorf("expected content 'hi', got %v", result["content"])
	}
	if result["node_id"] == nil || result["node_id"] == "" {
		t.Error("expected node_id to be set")
	}
}

func TestPrompt_MissingMessage(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/prompt", "application/json", jsonBody(`{"model":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	result := parseJSON(t, resp.Body)
	if result["error"] != "message is required" {
		t.Errorf("expected 'message is required' error, got %v", result["error"])
	}
}

func TestPrompt_InvalidJSON(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/prompt", "application/json", jsonBody(`{invalid`))
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

func TestPrompt_EmptyBody(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/prompt", "application/json", jsonBody(""))
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d", resp.StatusCode)
	}
}

func TestPrompt_WrongMethod(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/prompt")
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET on POST endpoint, got %d", resp.StatusCode)
	}
}

func TestPrompt_MultipleRequests(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	for i := 0; i < 3; i++ {
		resp, err := http.Post(ts.URL+"/prompt", "application/json",
			jsonBody(fmt.Sprintf(`{"message":"msg-%d","model":"test"}`, i)))
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, resp.StatusCode)
		}
		drainBody(t, resp)
	}
}

// --- Streaming SSE Tests ---

func TestPrompt_StreamingSSE(t *testing.T) {
	ts := testServerWithProvider(t, &streamingProvider{})
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/prompt", "application/json",
		jsonBody(`{"message":"hello","model":"test","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %s", ct)
	}

	cc := resp.Header.Get("Cache-Control")
	if cc != "no-cache" {
		t.Errorf("expected no-cache, got %s", cc)
	}

	// Read SSE events
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var events []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			events = append(events, strings.TrimPrefix(line, "data: "))
		}
	}

	if len(events) < 3 {
		t.Errorf("expected at least 3 SSE events (2 content + 1 done), got %d", len(events))
	}

	// Verify we got the content chunks and done event
	var foundContent strings.Builder
	var foundDone bool
	for _, evt := range events {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(evt), &parsed); err != nil {
			t.Errorf("failed to parse SSE event: %v (data: %s)", err, evt)
			continue
		}
		switch parsed["type"] {
		case "delta":
			foundContent.WriteString(parsed["content"].(string))
		case "done":
			foundDone = true
		}
	}

	if foundContent.String() != "hello world" {
		t.Errorf("expected content 'hello world', got '%s'", foundContent.String())
	}
	if !foundDone {
		t.Error("expected done event in stream")
	}
}

func TestPrompt_StreamingMultipleContentDeltas(t *testing.T) {
	ts := testServerWithProvider(t, &streamingProvider{})
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/prompt", "application/json",
		jsonBody(`{"message":"chunk test","model":"test","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var deltaCount int
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var parsed map[string]interface{}
		data := strings.TrimPrefix(line, "data: ")
		if err := json.Unmarshal([]byte(data), &parsed); err != nil {
			continue
		}
		if parsed["type"] == "delta" {
			deltaCount++
		}
	}

	// streamingProvider sends 3 content chunks
	if deltaCount != 3 {
		t.Errorf("expected 3 delta events, got %d", deltaCount)
	}
}

// --- Error Handling Tests ---

func TestPrompt_ProviderError(t *testing.T) {
	ts := testServerWithProvider(t, &errorProvider{err: fmt.Errorf("provider unavailable")})
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/prompt", "application/json",
		jsonBody(`{"message":"hello","model":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 for provider error, got %d", resp.StatusCode)
	}

	result := parseJSON(t, resp.Body)
	if !strings.Contains(result["error"].(string), "provider unavailable") {
		t.Errorf("expected error to contain 'provider unavailable', got %v", result["error"])
	}
}

func TestPrompt_StreamingProviderError(t *testing.T) {
	ts := testServerWithProvider(t, &errorStreamProvider{})
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/prompt", "application/json",
		jsonBody(`{"message":"hello","model":"test","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Streaming starts with 200, then error is sent as SSE event
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for streaming, got %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var foundError bool
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var parsed map[string]interface{}
		data := strings.TrimPrefix(line, "data: ")
		if err := json.Unmarshal([]byte(data), &parsed); err != nil {
			continue
		}
		if parsed["type"] == "error" {
			foundError = true
			if !strings.Contains(parsed["error"].(string), "rate limit") {
				t.Errorf("expected rate limit error, got %v", parsed["error"])
			}
		}
	}

	if !foundError {
		t.Error("expected error event in SSE stream")
	}
}

func TestPrompt_MethodNotAllowed(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/prompt", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
}

// --- Authentication Error Tests ---

func TestAuth_MissingKey_Returns401(t *testing.T) {
	ts := testServerWithAPIKey(t, &mockProv{}, "my-secret-key")
	defer ts.Close()

	// No auth header
	resp, err := http.Post(ts.URL+"/prompt", "application/json",
		jsonBody(`{"message":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	result := parseJSON(t, resp.Body)
	if result["error"] != "unauthorized" {
		t.Errorf("expected 'unauthorized' error, got %v", result["error"])
	}
}

func TestAuth_WrongKey_Returns401(t *testing.T) {
	ts := testServerWithAPIKey(t, &mockProv{}, "correct-key")
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/prompt",
		jsonBody(`{"message":"hello","model":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-key")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong key, got %d", resp.StatusCode)
	}
}

func TestAuth_BearerPrefix(t *testing.T) {
	ts := testServerWithAPIKey(t, &mockProv{}, "test-token")
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/prompt",
		jsonBody(`{"message":"hello","model":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with Bearer token, got %d", resp.StatusCode)
	}
}

func TestAuth_XAPIKey(t *testing.T) {
	ts := testServerWithAPIKey(t, &mockProv{}, "test-token")
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/prompt",
		jsonBody(`{"message":"hello","model":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with X-API-Key, got %d", resp.StatusCode)
	}
}

func TestAuth_ProtectedEndpoints_AllRequireAuth(t *testing.T) {
	ts := testServerWithAPIKey(t, &mockProv{}, "secret")
	defer ts.Close()

	protectedEndpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"POST", "/prompt", `{"message":"hi"}`},
		{"GET", "/nodes", ""},
		{"GET", "/nodes/test-id", ""},
		{"GET", "/nodes/test-id/tree", ""},
		{"DELETE", "/nodes/test-id", ""},
		{"GET", "/api/usage", ""},
		{"GET", "/api/costs", ""},
		{"GET", "/api/health/providers", ""},
	}

	for _, ep := range protectedEndpoints {
		var body io.Reader
		if ep.body != "" {
			body = jsonBody(ep.body)
		}
		req, _ := http.NewRequestWithContext(context.Background(), ep.method, ts.URL+ep.path, body)
		if ep.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", ep.method, ep.path, err)
		}
		drainBody(t, resp)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", ep.method, ep.path, resp.StatusCode)
		}
	}
}

func TestAuth_HealthEndpoint_NotProtected(t *testing.T) {
	ts := testServerWithAPIKey(t, &mockProv{}, "secret")
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health should not require auth, got %d", resp.StatusCode)
	}
}

// --- Node Management Integration Tests ---

func TestNodes_CreatePromptAndGetNode(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	// Create a conversation via prompt
	resp, err := http.Post(ts.URL+"/prompt", "application/json",
		jsonBody(`{"message":"test conversation","model":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	nodeID := result["node_id"].(string)
	if nodeID == "" {
		t.Fatal("expected node_id")
	}

	// Retrieve the node
	resp, err = http.Get(ts.URL + "/nodes/" + nodeID)
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestNodes_ListAfterMultiplePrompts(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	// Create multiple conversations
	for i := 0; i < 3; i++ {
		resp, err := http.Post(ts.URL+"/prompt", "application/json",
			jsonBody(fmt.Sprintf(`{"message":"conv-%d","model":"test"}`, i)))
		if err != nil {
			t.Fatal(err)
		}
		drainBody(t, resp)
	}

	// List nodes
	resp, err := http.Get(ts.URL + "/nodes")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var nodes []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if len(nodes) != 3 {
		t.Errorf("expected 3 root nodes, got %d", len(nodes))
	}
}

func TestNodes_DeleteNode(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	// Create a conversation
	resp, err := http.Post(ts.URL+"/prompt", "application/json",
		jsonBody(`{"message":"to delete","model":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	_ = resp.Body.Close()
	nodeID := result["node_id"].(string)

	// Delete it
	req, _ := http.NewRequestWithContext(context.Background(), "DELETE", ts.URL+"/nodes/"+nodeID, nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	drainBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Verify it's gone
	resp, err = http.Get(ts.URL + "/nodes/" + nodeID)
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
}

func TestNodes_GetNonExistent_Returns404(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/nodes/nonexistent-id-12345")
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent node, got %d", resp.StatusCode)
	}
}

// --- Alias Integration Tests ---

func TestAlias_CreateAndGet(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	// Create a conversation
	resp, err := http.Post(ts.URL+"/prompt", "application/json",
		jsonBody(`{"message":"alias test","model":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	_ = resp.Body.Close()
	nodeID := result["node_id"].(string)

	// Create alias
	req, _ := http.NewRequestWithContext(context.Background(), "PUT",
		ts.URL+"/nodes/"+nodeID+"/aliases/my-alias", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	drainBody(t, resp)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for alias creation, got %d", resp.StatusCode)
	}

	// Get node by alias
	resp, err = http.Get(ts.URL + "/nodes/my-alias")
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for alias lookup, got %d", resp.StatusCode)
	}
}

func TestAlias_Delete(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	// Create a conversation
	resp, err := http.Post(ts.URL+"/prompt", "application/json",
		jsonBody(`{"message":"alias delete test","model":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	_ = resp.Body.Close()
	nodeID := result["node_id"].(string)

	// Create alias
	req, _ := http.NewRequestWithContext(context.Background(), "PUT",
		ts.URL+"/nodes/"+nodeID+"/aliases/temp-alias", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	drainBody(t, resp)

	// Delete alias
	req, _ = http.NewRequestWithContext(context.Background(), "DELETE",
		ts.URL+"/aliases/temp-alias", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	drainBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for alias deletion, got %d", resp.StatusCode)
	}
}

// --- PromptFrom Endpoint Tests ---

func TestPromptFrom_ContinueConversation(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	// Start a conversation
	resp, err := http.Post(ts.URL+"/prompt", "application/json",
		jsonBody(`{"message":"start","model":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	_ = resp.Body.Close()
	nodeID := result["node_id"].(string)

	// Continue from the assistant node
	resp, err = http.Post(ts.URL+"/nodes/"+nodeID+"/prompt", "application/json",
		jsonBody(`{"message":"continue","model":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for PromptFrom, got %d", resp.StatusCode)
	}

	var contResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&contResult); err != nil {
		t.Fatal(err)
	}

	if contResult["node_id"] == nil || contResult["node_id"] == "" {
		t.Error("expected node_id in continuation response")
	}
}

func TestPromptFrom_StreamingContinuation(t *testing.T) {
	ts := testServerWithProvider(t, &streamingProvider{})
	defer ts.Close()

	// Start a conversation
	resp, err := http.Post(ts.URL+"/prompt", "application/json",
		jsonBody(`{"message":"start","model":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	_ = resp.Body.Close()
	nodeID := result["node_id"].(string)

	// Continue with streaming
	resp, err = http.Post(ts.URL+"/nodes/"+nodeID+"/prompt", "application/json",
		jsonBody(`{"message":"continue","model":"test","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %s", ct)
	}

	// Verify we get SSE events
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var eventCount int
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			eventCount++
		}
	}

	if eventCount < 2 {
		t.Errorf("expected at least 2 SSE events, got %d", eventCount)
	}
}

// --- Rate Limit / 429 Simulation Tests ---

// rateLimitProvider simulates a 429-like behavior by returning an error
// that would typically come from a rate-limited provider.
type rateLimitProvider struct {
	callCount int
	limit     int
}

func (r *rateLimitProvider) Name() string                 { return "ratelimit" }
func (r *rateLimitProvider) Ping(_ context.Context) error { return nil }
func (r *rateLimitProvider) Chat(_ context.Context, _ []client.EyrieMessage, _ client.ChatOptions) (*client.EyrieResponse, error) {
	r.callCount++
	if r.callCount > r.limit {
		return nil, fmt.Errorf("429 Too Many Requests: rate limit exceeded")
	}
	return &client.EyrieResponse{Content: "ok", FinishReason: "end_turn", Usage: &client.EyrieUsage{CompletionTokens: 1}}, nil
}

func (r *rateLimitProvider) StreamChat(_ context.Context, _ []client.EyrieMessage, _ client.ChatOptions) (*client.StreamResult, error) {
	r.callCount++
	if r.callCount > r.limit {
		return nil, fmt.Errorf("429 Too Many Requests: rate limit exceeded")
	}
	ch := make(chan client.EyrieStreamEvent, 2)
	ch <- client.EyrieStreamEvent{Type: "content", Content: "ok"}
	ch <- client.EyrieStreamEvent{Type: "done", StopReason: "end_turn", Usage: &client.EyrieUsage{CompletionTokens: 1}}
	close(ch)
	return &client.StreamResult{Events: ch}, nil
}

func TestPrompt_RateLimitSimulation(t *testing.T) {
	provider := &rateLimitProvider{limit: 1}
	ts := testServerWithProvider(t, provider)
	defer ts.Close()

	// First request succeeds
	resp, err := http.Post(ts.URL+"/prompt", "application/json",
		jsonBody(`{"message":"first","model":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	drainBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", resp.StatusCode)
	}

	// Second request hits rate limit
	resp, err = http.Post(ts.URL+"/prompt", "application/json",
		jsonBody(`{"message":"second","model":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	// The server returns 500 since the provider error propagates
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("second request: expected 500, got %d", resp.StatusCode)
	}

	result := parseJSON(t, resp.Body)
	errMsg := result["error"].(string)
	if !strings.Contains(errMsg, "rate limit") {
		t.Errorf("expected rate limit error, got %v", errMsg)
	}
}

// --- Provider Error 500 Simulation ---

func TestPrompt_InternalServerError(t *testing.T) {
	provider := &errorProvider{err: fmt.Errorf("internal server error: model overloaded")}
	ts := testServerWithProvider(t, provider)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/prompt", "application/json",
		jsonBody(`{"message":"hello","model":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}

	result := parseJSON(t, resp.Body)
	if !strings.Contains(result["error"].(string), "model overloaded") {
		t.Errorf("expected error to contain 'model overloaded', got %v", result["error"])
	}
}

// --- Content-Type Validation ---

func TestPrompt_WrongContentType(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	// Go's JSON decoder does not enforce Content-Type; the server accepts
	// valid JSON regardless of Content-Type header. Verify it still works.
	resp, err := http.Post(ts.URL+"/prompt", "text/plain", jsonBody(`{"message":"hello","model":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (server parses JSON regardless of Content-Type), got %d", resp.StatusCode)
	}
}

// --- Concurrent Request Tests ---

func TestPrompt_ConcurrentRequests(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	const numRequests = 10
	errs := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			resp, err := http.Post(ts.URL+"/prompt", "application/json",
				jsonBody(fmt.Sprintf(`{"message":"concurrent-%d","model":"test"}`, idx)))
			if err != nil {
				errs <- fmt.Errorf("request %d: %v", idx, err)
				return
			}
			drainBody(t, resp)
			if resp.StatusCode != http.StatusOK {
				errs <- fmt.Errorf("request %d: expected 200, got %d", idx, resp.StatusCode)
				return
			}
			errs <- nil
		}(i)
	}

	for i := 0; i < numRequests; i++ {
		if err := <-errs; err != nil {
			t.Error(err)
		}
	}
}

// --- Tree Endpoint Tests ---

func TestTree_ReturnsSubtree(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	// Create a conversation
	resp, err := http.Post(ts.URL+"/prompt", "application/json",
		jsonBody(`{"message":"tree root","model":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	_ = resp.Body.Close()
	assistantNodeID := result["node_id"].(string)

	// Get the root user node from /nodes
	resp, err = http.Get(ts.URL + "/nodes")
	if err != nil {
		t.Fatal(err)
	}
	var nodes []map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&nodes)
	_ = resp.Body.Close()

	if len(nodes) == 0 {
		t.Fatal("expected at least 1 root node")
	}
	rootNodeID := nodes[0]["id"].(string)

	// Verify user node != assistant node
	if rootNodeID == assistantNodeID {
		t.Error("root node should differ from assistant node")
	}

	// Get tree from root
	resp, err = http.Get(ts.URL + "/nodes/" + rootNodeID + "/tree")
	if err != nil {
		t.Fatal(err)
	}
	defer drainBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var treeNodes []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&treeNodes); err != nil {
		t.Fatal(err)
	}

	// Should contain the user node and the assistant node
	if len(treeNodes) < 2 {
		t.Errorf("expected at least 2 nodes in tree, got %d", len(treeNodes))
	}
}
