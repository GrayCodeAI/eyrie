package conversation

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/GrayCodeAI/eyrie/client"
	"github.com/GrayCodeAI/eyrie/storage"
)

type mockStreamProvider struct{}

func (m *mockStreamProvider) Name() string                 { return "mock" }
func (m *mockStreamProvider) Ping(_ context.Context) error { return nil }
func (m *mockStreamProvider) Chat(_ context.Context, _ []client.EyrieMessage, _ client.ChatOptions) (*client.EyrieResponse, error) {
	return &client.EyrieResponse{Content: "hello", FinishReason: "end_turn", Usage: &client.EyrieUsage{CompletionTokens: 5}}, nil
}

func (m *mockStreamProvider) StreamChat(_ context.Context, _ []client.EyrieMessage, _ client.ChatOptions) (*client.StreamResult, error) {
	ch := make(chan client.EyrieStreamEvent, 3)
	ch <- client.EyrieStreamEvent{Type: "content", Content: "hello"}
	ch <- client.EyrieStreamEvent{Type: "done", StopReason: "end_turn", Usage: &client.EyrieUsage{CompletionTokens: 5}}
	close(ch)
	return &client.StreamResult{Events: ch}, nil
}

func testEngine(t *testing.T) *Engine {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return New(store, &mockStreamProvider{})
}

func TestPromptCreatesNodes(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()

	events, err := e.Prompt(ctx, "hi", PromptOpts{Model: "test"})
	if err != nil {
		t.Fatal(err)
	}
	var gotContent string
	var nodeID string
	for ev := range events {
		switch ev.Type {
		case EventDelta:
			gotContent += ev.Content
		case EventDone:
			nodeID = ev.NodeID
		}
	}
	if gotContent != "hello" {
		t.Errorf("expected 'hello', got %q", gotContent)
	}
	if nodeID == "" {
		t.Error("expected node_id in done event")
	}

	convos, err := e.ListConversations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(convos) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(convos))
	}
}

func TestPromptFromBranches(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()

	events, _ := e.Prompt(ctx, "first", PromptOpts{Model: "test"})
	var firstNodeID string
	for ev := range events {
		if ev.Type == EventDone {
			firstNodeID = ev.NodeID
		}
	}

	events2, err := e.PromptFrom(ctx, firstNodeID, "branch", PromptOpts{Model: "test"})
	if err != nil {
		t.Fatal(err)
	}
	var secondNodeID string
	for ev := range events2 {
		if ev.Type == EventDone {
			secondNodeID = ev.NodeID
		}
	}
	if secondNodeID == "" {
		t.Error("expected second node")
	}
	if secondNodeID == firstNodeID {
		t.Error("expected different node IDs for branch")
	}
}

func TestResolveNode(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()

	events, _ := e.Prompt(ctx, "test resolve", PromptOpts{Model: "test"})
	var nodeID string
	for ev := range events {
		if ev.Type == EventDone {
			nodeID = ev.NodeID
		}
	}

	got, err := e.ResolveNode(ctx, nodeID[:8])
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != nodeID {
		t.Errorf("expected %s, got %s", nodeID, got.ID)
	}
}
