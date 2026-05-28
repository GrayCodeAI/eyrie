package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- Storage creation / configuration ---

func TestOpenCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database file not created: %v", err)
	}
}

func TestOpenInMemory(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	node := &Node{ID: "mem1", NodeType: NodeTypeUser, Content: "in-memory", Sequence: 1}
	if err := s.CreateNode(ctx, node); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetNode(ctx, "mem1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "in-memory" {
		t.Errorf("expected 'in-memory', got %q", got.Content)
	}
}

func TestOpenReusesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reuse.db")

	s1, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	s1.CreateNode(ctx, &Node{ID: "r1", NodeType: NodeTypeUser, Content: "first", Sequence: 1})
	s1.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	got, err := s2.GetNode(ctx, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "first" {
		t.Errorf("expected 'first', got %q", got.Content)
	}
}

func TestClose(t *testing.T) {
	s := testStore(t)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

// --- Store / Load / Delete ---

func TestCreateNodeAutoTimestamp(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	before := time.Now().Add(-time.Second)
	node := &Node{ID: "ts1", NodeType: NodeTypeUser, Content: "auto-ts", Sequence: 1}
	if err := s.CreateNode(ctx, node); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetNode(ctx, "ts1")
	if err != nil {
		t.Fatal(err)
	}
	if got.CreatedAt.Before(before) {
		t.Errorf("expected auto-set timestamp >= %v, got %v", before, got.CreatedAt)
	}
}

func TestCreateNodeExplicitTimestamp(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	node := &Node{ID: "ts2", NodeType: NodeTypeUser, Content: "explicit-ts", Sequence: 1, CreatedAt: ts}
	if err := s.CreateNode(ctx, node); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetNode(ctx, "ts2")
	if err != nil {
		t.Fatal(err)
	}
	if !got.CreatedAt.Equal(ts) {
		t.Errorf("expected %v, got %v", ts, got.CreatedAt)
	}
}

func TestCreateNodeWithMetadata(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	meta := json.RawMessage(`{"key":"value","num":42}`)
	node := &Node{ID: "meta1", NodeType: NodeTypeUser, Content: "has-meta", Sequence: 1, Metadata: meta}
	if err := s.CreateNode(ctx, node); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetNode(ctx, "meta1")
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Metadata) != string(meta) {
		t.Errorf("metadata mismatch: got %s, want %s", got.Metadata, meta)
	}
}

func TestUpdateNode(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	node := &Node{ID: "u1", NodeType: NodeTypeUser, Content: "original", Sequence: 1, Provider: "openai", Model: "gpt-4"}
	if err := s.CreateNode(ctx, node); err != nil {
		t.Fatal(err)
	}

	node.Content = "updated"
	node.Model = "gpt-4o"
	node.TokensIn = 100
	node.TokensOut = 50
	node.Title = "new title"
	if err := s.UpdateNode(ctx, node); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetNode(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "updated" {
		t.Errorf("expected content 'updated', got %q", got.Content)
	}
	if got.Model != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %q", got.Model)
	}
	if got.TokensIn != 100 || got.TokensOut != 50 {
		t.Errorf("expected tokens 100/50, got %d/%d", got.TokensIn, got.TokensOut)
	}
	if got.Title != "new title" {
		t.Errorf("expected title 'new title', got %q", got.Title)
	}
}

func TestUpdateNodeMetadata(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	node := &Node{ID: "um1", NodeType: NodeTypeUser, Content: "test", Sequence: 1}
	s.CreateNode(ctx, node)

	node.Metadata = json.RawMessage(`{"updated":true}`)
	if err := s.UpdateNode(ctx, node); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetNode(ctx, "um1")
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Metadata) != `{"updated":true}` {
		t.Errorf("metadata not updated: %s", got.Metadata)
	}
}

func TestGetNodeChildren(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "parent", NodeType: NodeTypeUser, Content: "p", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "c1", ParentID: "parent", RootID: "parent", NodeType: NodeTypeAssistant, Content: "c1", Sequence: 2})
	s.CreateNode(ctx, &Node{ID: "c2", ParentID: "parent", RootID: "parent", NodeType: NodeTypeAssistant, Content: "c2", Sequence: 3})
	s.CreateNode(ctx, &Node{ID: "c3", ParentID: "parent", RootID: "parent", NodeType: NodeTypeUser, Content: "c3", Sequence: 4})

	children, err := s.GetNodeChildren(ctx, "parent")
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(children))
	}
	// Should be ordered by sequence.
	if children[0].ID != "c1" || children[1].ID != "c2" || children[2].ID != "c3" {
		t.Errorf("unexpected child order: %s, %s, %s", children[0].ID, children[1].ID, children[2].ID)
	}
}

func TestGetNodeChildrenEmpty(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "leaf", NodeType: NodeTypeUser, Content: "leaf", Sequence: 1})

	children, err := s.GetNodeChildren(ctx, "leaf")
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 0 {
		t.Errorf("expected 0 children, got %d", len(children))
	}
}

func TestDeleteAndVerifyGone(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "del1", NodeType: NodeTypeUser, Content: "delete me", Sequence: 1})

	if err := s.DeleteNode(ctx, "del1"); err != nil {
		t.Fatal(err)
	}
	_, err := s.GetNode(ctx, "del1")
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestDeleteNonExistent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	// Deleting a non-existent node should not error (0 rows affected).
	if err := s.DeleteNode(ctx, "does-not-exist"); err != nil {
		t.Fatalf("unexpected error deleting non-existent node: %v", err)
	}
}

func TestCreateAllNodeTypes(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	types := []NodeType{
		NodeTypeUser,
		NodeTypeAssistant,
		NodeTypeSystem,
		NodeTypeToolCall,
		NodeTypeToolResult,
	}
	for i, nt := range types {
		node := &Node{ID: string(nt), NodeType: nt, Content: string(nt), Sequence: i + 1}
		if err := s.CreateNode(ctx, node); err != nil {
			t.Fatalf("failed to create node type %s: %v", nt, err)
		}
	}
	for _, nt := range types {
		got, err := s.GetNode(ctx, string(nt))
		if err != nil {
			t.Fatalf("failed to get node type %s: %v", nt, err)
		}
		if got.NodeType != nt {
			t.Errorf("expected node type %s, got %s", nt, got.NodeType)
		}
	}
}

func TestTokenFields(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	node := &Node{
		ID:                  "tok1",
		NodeType:            NodeTypeAssistant,
		Content:             "response",
		Sequence:            1,
		TokensIn:            100,
		TokensOut:           200,
		TokensCacheRead:     50,
		TokensCacheCreation: 25,
		TokensReasoning:     75,
		LatencyMs:           1500,
		StopReason:          "end_turn",
		OutputGroupID:       "grp1",
	}
	s.CreateNode(ctx, node)
	got, err := s.GetNode(ctx, "tok1")
	if err != nil {
		t.Fatal(err)
	}
	if got.TokensIn != 100 || got.TokensOut != 200 || got.TokensCacheRead != 50 || got.TokensCacheCreation != 25 || got.TokensReasoning != 75 {
		t.Errorf("token fields mismatch: %+v", got)
	}
	if got.LatencyMs != 1500 {
		t.Errorf("expected latency 1500, got %d", got.LatencyMs)
	}
	if got.StopReason != "end_turn" {
		t.Errorf("expected stop reason 'end_turn', got %q", got.StopReason)
	}
	if got.OutputGroupID != "grp1" {
		t.Errorf("expected output group 'grp1', got %q", got.OutputGroupID)
	}
}

// --- Key validation / constraint tests ---

func TestDuplicateIDRejected(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "dup", NodeType: NodeTypeUser, Content: "first", Sequence: 1})
	err := s.CreateNode(ctx, &Node{ID: "dup", NodeType: NodeTypeUser, Content: "second", Sequence: 2})
	if err == nil {
		t.Error("expected error for duplicate ID, got nil")
	}
}

func TestForeignKeyConstraint(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	err := s.CreateNode(ctx, &Node{ID: "orphan", ParentID: "nonexistent", NodeType: NodeTypeUser, Content: "orphan", Sequence: 1})
	if err == nil {
		t.Error("expected foreign key violation for non-existent parent, got nil")
	}
}

// --- Error cases ---

func TestGetNonExistentNode(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	_, err := s.GetNode(ctx, "no-such-id")
	if err == nil {
		t.Error("expected error for non-existent node, got nil")
	}
}

func TestGetNodeByPrefixNoMatch(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "abc123", NodeType: NodeTypeUser, Content: "test", Sequence: 1})
	_, err := s.GetNodeByPrefix(ctx, "zzz")
	if err == nil {
		t.Error("expected error for non-matching prefix, got nil")
	}
}

func TestGetNodeByPrefixAmbiguous(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "abcdef", NodeType: NodeTypeUser, Content: "first", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "abcghi", NodeType: NodeTypeUser, Content: "second", Sequence: 2})
	// Prefix "abc" matches both; should return one (LIMIT 1).
	got, err := s.GetNodeByPrefix(ctx, "abc")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Error("expected a node for ambiguous prefix, got nil")
	}
}

func TestGetNonExistentAlias(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	_, err := s.GetNodeByAlias(ctx, "no-such-alias")
	if err == nil {
		t.Error("expected error for non-existent alias, got nil")
	}
}

func TestAliasReplace(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "n1", NodeType: NodeTypeUser, Content: "first", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "n2", NodeType: NodeTypeUser, Content: "second", Sequence: 2})

	s.CreateAlias(ctx, "myalias", "n1")
	s.CreateAlias(ctx, "myalias", "n2") // INSERT OR REPLACE

	got, err := s.GetNodeByAlias(ctx, "myalias")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "n2" {
		t.Errorf("expected alias to point to n2 after replace, got %s", got.ID)
	}
}

func TestAliasCascadeDelete(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "an1", NodeType: NodeTypeUser, Content: "test", Sequence: 1})
	s.CreateAlias(ctx, "delalias", "an1")

	s.DeleteNode(ctx, "an1")
	_, err := s.GetNodeByAlias(ctx, "delalias")
	if err == nil {
		t.Error("expected alias to be cascade-deleted when node is deleted")
	}
}

func TestListAliasesMultiple(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "mn1", NodeType: NodeTypeUser, Content: "multi", Sequence: 1})
	s.CreateAlias(ctx, "alias1", "mn1")
	s.CreateAlias(ctx, "alias2", "mn1")
	s.CreateAlias(ctx, "alias3", "mn1")

	aliases, err := s.ListAliases(ctx, "mn1")
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != 3 {
		t.Errorf("expected 3 aliases, got %d", len(aliases))
	}
}

func TestListAliasesEmpty(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "noalias", NodeType: NodeTypeUser, Content: "no alias", Sequence: 1})

	aliases, err := s.ListAliases(ctx, "noalias")
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != 0 {
		t.Errorf("expected 0 aliases, got %d", len(aliases))
	}
}

func TestIndexToolIDsAndOrphans(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "tn1", NodeType: NodeTypeAssistant, Content: "tool node", Sequence: 1})
	s.IndexToolIDs(ctx, "tn1", []string{"tid-a", "tid-b", "tid-c"}, "use")
	s.IndexToolIDs(ctx, "tn1", []string{"tid-a"}, "result")

	orphans, err := s.GetOrphanedToolUses(ctx, []string{"tn1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans["tn1"]) != 2 {
		t.Errorf("expected 2 orphaned tool uses, got %d", len(orphans["tn1"]))
	}
}

func TestOrphanedToolUsesEmptyInput(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	result, err := s.GetOrphanedToolUses(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil result for empty input, got %v", result)
	}
}

func TestGetSubtreeSingleNode(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "only", NodeType: NodeTypeUser, Content: "only", Sequence: 1})

	nodes, err := s.GetSubtree(ctx, "only")
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Errorf("expected 1 node in subtree, got %d", len(nodes))
	}
}

func TestGetAncestorsSingleNode(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "root", NodeType: NodeTypeUser, Content: "root", Sequence: 1})

	ancestors, err := s.GetAncestors(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	if len(ancestors) != 1 || ancestors[0].ID != "root" {
		t.Errorf("expected [root], got %v", ancestors)
	}
}

func TestCreateNodeWithAllFields(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	node := &Node{
		ID:                  "full",
		ParentID:            "",
		RootID:              "",
		Sequence:            1,
		NodeType:            NodeTypeAssistant,
		Content:             "full node",
		Provider:            "anthropic",
		Model:               "claude-sonnet-4-20250514",
		TokensIn:            500,
		TokensOut:           300,
		TokensCacheRead:     100,
		TokensCacheCreation: 50,
		TokensReasoning:     200,
		LatencyMs:           2000,
		StopReason:          "max_tokens",
		OutputGroupID:       "group-1",
		Status:              "complete",
		Title:               "Full Test",
		SystemPrompt:        "You are a test.",
		Metadata:            json.RawMessage(`{"test":true}`),
		CreatedAt:           time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := s.CreateNode(ctx, node); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetNode(ctx, "full")
	if err != nil {
		t.Fatal(err)
	}
	if got.Provider != "anthropic" {
		t.Errorf("expected provider 'anthropic', got %q", got.Provider)
	}
	if got.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected model 'claude-sonnet-4-20250514', got %q", got.Model)
	}
	if got.Status != "complete" {
		t.Errorf("expected status 'complete', got %q", got.Status)
	}
	if got.SystemPrompt != "You are a test." {
		t.Errorf("expected system prompt 'You are a test.', got %q", got.SystemPrompt)
	}
	if got.Title != "Full Test" {
		t.Errorf("expected title 'Full Test', got %q", got.Title)
	}
}

func TestDAGOperations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dag_test.db")
	dag, err := NewDAG(path, "test-session")
	if err != nil {
		t.Fatal(err)
	}
	defer dag.Close()

	ctx := context.Background()
	// Append root.
	root, err := dag.Append(ctx, "", "user", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if root.Role != "user" || root.Content != "hello" {
		t.Errorf("unexpected root: %+v", root)
	}

	// Append child.
	child, err := dag.Append(ctx, root.ID, "assistant", "hi there")
	if err != nil {
		t.Fatal(err)
	}
	if child.ParentID != root.ID {
		t.Errorf("expected parent %s, got %s", root.ID, child.ParentID)
	}

	// Head should be the child.
	head, err := dag.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if head.ID != child.ID {
		t.Errorf("expected head %s, got %s", child.ID, head.ID)
	}

	// History from child should return both nodes.
	history, err := dag.History(ctx, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 nodes in history, got %d", len(history))
	}
	if history[0].ID != root.ID || history[1].ID != child.ID {
		t.Errorf("unexpected history order: %s, %s", history[0].ID, history[1].ID)
	}

	// Branches from root should return child.
	branches, err := dag.Branches(ctx, root.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 1 || branches[0].ID != child.ID {
		t.Errorf("expected 1 branch (child), got %d", len(branches))
	}
}

func TestDAGFork(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fork_test.db")
	dag, err := NewDAG(path, "fork-session")
	if err != nil {
		t.Fatal(err)
	}
	defer dag.Close()

	ctx := context.Background()
	root, _ := dag.Append(ctx, "", "user", "original")
	child, _ := dag.Append(ctx, root.ID, "assistant", "response")

	// Fork from child.
	fork, err := dag.Fork(ctx, child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fork.ParentID != child.ParentID {
		t.Errorf("expected fork parent %s, got %s", child.ParentID, fork.ParentID)
	}
	if fork.Content != child.Content {
		t.Errorf("expected fork content %q, got %q", child.Content, fork.Content)
	}
}

func TestDAGPrune(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prune_test.db")
	dag, err := NewDAG(path, "prune-session")
	if err != nil {
		t.Fatal(err)
	}
	defer dag.Close()

	ctx := context.Background()
	root, _ := dag.Append(ctx, "", "user", "root")
	dag.Append(ctx, root.ID, "assistant", "child")

	if err := dag.Prune(ctx, root.ID); err != nil {
		t.Fatal(err)
	}

	// Root should be gone (child cascade-deleted).
	_, err = dag.History(ctx, root.ID)
	if err == nil {
		t.Error("expected error after pruning root")
	}
}

func TestDAGSetHead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "head_test.db")
	dag, err := NewDAG(path, "head-session")
	if err != nil {
		t.Fatal(err)
	}
	defer dag.Close()

	ctx := context.Background()
	root, _ := dag.Append(ctx, "", "user", "root")
	dag.Append(ctx, root.ID, "assistant", "child")

	if err := dag.SetHead(ctx, root.ID); err != nil {
		t.Fatal(err)
	}
	head, err := dag.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if head.ID != root.ID {
		t.Errorf("expected head %s, got %s", root.ID, head.ID)
	}

	// SetHead to non-existent should error.
	if err := dag.SetHead(ctx, "nonexistent"); err == nil {
		t.Error("expected error for SetHead with non-existent node")
	}
}

func TestDAGHeadBeforeAppend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty_dag.db")
	dag, err := NewDAG(path, "empty-session")
	if err != nil {
		t.Fatal(err)
	}
	defer dag.Close()

	_, err = dag.Head(context.Background())
	if err == nil {
		t.Error("expected error calling Head on empty DAG")
	}
}

func TestDAGAppendInvalidParent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid_parent.db")
	dag, err := NewDAG(path, "invalid-session")
	if err != nil {
		t.Fatal(err)
	}
	defer dag.Close()

	_, err = dag.Append(context.Background(), "nonexistent", "user", "content")
	if err == nil {
		t.Error("expected error appending with non-existent parent")
	}
}

func TestDAGFromStore(t *testing.T) {
	s := testStore(t)
	dag := NewDAGFromStore(s, "from-store-session")
	ctx := context.Background()

	root, err := dag.Append(ctx, "", "user", "test")
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.GetNode(ctx, root.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "test" {
		t.Errorf("expected 'test', got %q", got.Content)
	}
	if got.RootID != "from-store-session" {
		t.Errorf("expected root_id 'from-store-session', got %q", got.RootID)
	}
}
