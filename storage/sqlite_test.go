package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *SQLiteStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateAndGetNode(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	node := &Node{ID: "n1", NodeType: NodeTypeUser, Content: "hello", Sequence: 1}
	if err := s.CreateNode(ctx, node); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetNode(ctx, "n1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "hello" || got.NodeType != NodeTypeUser {
		t.Errorf("got %+v", got)
	}
}

func TestGetSubtree(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "root", NodeType: NodeTypeUser, Content: "root", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "child", ParentID: "root", RootID: "root", NodeType: NodeTypeAssistant, Content: "child", Sequence: 2})
	s.CreateNode(ctx, &Node{ID: "grandchild", ParentID: "child", RootID: "root", NodeType: NodeTypeUser, Content: "grandchild", Sequence: 3})

	nodes, err := s.GetSubtree(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
}

func TestGetAncestors(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "a", NodeType: NodeTypeUser, Content: "a", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "b", ParentID: "a", RootID: "a", NodeType: NodeTypeAssistant, Content: "b", Sequence: 2})
	s.CreateNode(ctx, &Node{ID: "c", ParentID: "b", RootID: "a", NodeType: NodeTypeUser, Content: "c", Sequence: 3})

	ancestors, err := s.GetAncestors(ctx, "c")
	if err != nil {
		t.Fatal(err)
	}
	if len(ancestors) != 3 {
		t.Fatalf("expected 3 ancestors, got %d", len(ancestors))
	}
	if ancestors[0].ID != "a" {
		t.Errorf("expected root first, got %s", ancestors[0].ID)
	}
}

func TestDeleteNodeCascades(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "r", NodeType: NodeTypeUser, Content: "r", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "c1", ParentID: "r", RootID: "r", NodeType: NodeTypeAssistant, Content: "c1", Sequence: 2})

	if err := s.DeleteNode(ctx, "r"); err != nil {
		t.Fatal(err)
	}
	_, err := s.GetNode(ctx, "c1")
	if err == nil {
		t.Error("expected child to be cascade-deleted")
	}
}

func TestAliases(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "n1", NodeType: NodeTypeUser, Content: "hello", Sequence: 1})

	if err := s.CreateAlias(ctx, "start", "n1"); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetNodeByAlias(ctx, "start")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "n1" {
		t.Errorf("expected n1, got %s", got.ID)
	}

	aliases, err := s.ListAliases(ctx, "n1")
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != 1 || aliases[0].Alias != "start" {
		t.Errorf("unexpected aliases: %+v", aliases)
	}

	s.DeleteAlias(ctx, "start")
	_, err = s.GetNodeByAlias(ctx, "start")
	if err == nil {
		t.Error("expected alias to be deleted")
	}
}

func TestListRootNodes(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "r1", NodeType: NodeTypeUser, Content: "root1", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "r2", NodeType: NodeTypeUser, Content: "root2", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "child", ParentID: "r1", RootID: "r1", NodeType: NodeTypeAssistant, Content: "child", Sequence: 2})

	roots, err := s.ListRootNodes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(roots))
	}
}

func TestOrphanedToolUses(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "r", NodeType: NodeTypeUser, Content: "r", RootID: "r", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "a", ParentID: "r", RootID: "r", NodeType: NodeTypeAssistant, Content: "a", Sequence: 2})

	s.IndexToolIDs(ctx, "a", []string{"tool-1", "tool-2"}, "tool_use")
	s.IndexToolIDs(ctx, "r", []string{"tool-1"}, "tool_result")

	orphans, err := s.GetOrphanedToolUses(ctx, "r")
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 1 || orphans[0] != "tool-2" {
		t.Errorf("expected [tool-2], got %v", orphans)
	}
}

func TestNodeByPrefix(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	s.CreateNode(ctx, &Node{ID: "abc123def", NodeType: NodeTypeUser, Content: "test", Sequence: 1})

	got, err := s.GetNodeByPrefix(ctx, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "abc123def" {
		t.Errorf("expected abc123def, got %s", got.ID)
	}
}
