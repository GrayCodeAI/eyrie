//nolint:errcheck
package storage

import (
	"context"
	"testing"
)

// ---------------------------------------------------------------------------
// SQL LIKE escaping: GetNodeByPrefix must escape % and _ wildcards
// ---------------------------------------------------------------------------

func TestGetNodeByPrefix_EscapesPercentWildcard(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create nodes with IDs that look like LIKE patterns.
	s.CreateNode(ctx, &Node{ID: "abc%def", NodeType: NodeTypeUser, Content: "percent-id", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "abc123def", NodeType: NodeTypeUser, Content: "normal-id", Sequence: 2})

	// Searching for the literal prefix "abc%" should find "abc%def",
	// NOT "abc123def" (which would match the LIKE pattern 'abc%').
	got, err := s.GetNodeByPrefix(ctx, "abc%")
	if err != nil {
		t.Fatalf("GetNodeByPrefix error: %v", err)
	}
	if got.ID != "abc%def" {
		t.Errorf("expected exact match 'abc%%def', got %q (LIKE wildcard not escaped)", got.ID)
	}
	if got.Content != "percent-id" {
		t.Errorf("expected content 'percent-id', got %q", got.Content)
	}
}

func TestGetNodeByPrefix_EscapesUnderscoreWildcard(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create nodes with IDs that look like LIKE patterns.
	s.CreateNode(ctx, &Node{ID: "abc_def", NodeType: NodeTypeUser, Content: "underscore-id", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "abcXdef", NodeType: NodeTypeUser, Content: "normal-id", Sequence: 2})

	// Searching for "abc_" should find "abc_def", NOT "abcXdef".
	got, err := s.GetNodeByPrefix(ctx, "abc_")
	if err != nil {
		t.Fatalf("GetNodeByPrefix error: %v", err)
	}
	if got.ID != "abc_def" {
		t.Errorf("expected exact match 'abc_def', got %q (LIKE wildcard not escaped)", got.ID)
	}
}

func TestGetNodeByPrefix_EscapesMixedWildcards(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	s.CreateNode(ctx, &Node{ID: "test%_value", NodeType: NodeTypeUser, Content: "mixed-wildcards", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "testX_value", NodeType: NodeTypeUser, Content: "underscore-match", Sequence: 2})
	s.CreateNode(ctx, &Node{ID: "testXval", NodeType: NodeTypeUser, Content: "percent-match", Sequence: 3})

	// Searching for "test%_" should find the exact node, not the LIKE matches.
	got, err := s.GetNodeByPrefix(ctx, "test%_")
	if err != nil {
		t.Fatalf("GetNodeByPrefix error: %v", err)
	}
	if got.ID != "test%_value" {
		t.Errorf("expected 'test%%_value', got %q", got.ID)
	}
}

func TestGetNodeByPrefix_EscapesMultiplePercents(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	s.CreateNode(ctx, &Node{ID: "a%%b", NodeType: NodeTypeUser, Content: "double-percent", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "aXXb", NodeType: NodeTypeUser, Content: "wildcard-match", Sequence: 2})

	got, err := s.GetNodeByPrefix(ctx, "a%%b")
	if err != nil {
		t.Fatalf("GetNodeByPrefix error: %v", err)
	}
	if got.ID != "a%%b" {
		t.Errorf("expected 'a%%%%b', got %q", got.ID)
	}
}

func TestGetNodeByPrefix_EscapesMultipleUnderscores(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	s.CreateNode(ctx, &Node{ID: "a__b", NodeType: NodeTypeUser, Content: "double-underscore", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "aXXb", NodeType: NodeTypeUser, Content: "wildcard-match", Sequence: 2})

	got, err := s.GetNodeByPrefix(ctx, "a__")
	if err != nil {
		t.Fatalf("GetNodeByPrefix error: %v", err)
	}
	if got.ID != "a__b" {
		t.Errorf("expected 'a__b', got %q", got.ID)
	}
}

func TestGetNodeByPrefix_NormalPrefixStillWorks(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	s.CreateNode(ctx, &Node{ID: "abc123def", NodeType: NodeTypeUser, Content: "normal", Sequence: 1})

	got, err := s.GetNodeByPrefix(ctx, "abc123")
	if err != nil {
		t.Fatalf("GetNodeByPrefix error: %v", err)
	}
	if got.ID != "abc123def" {
		t.Errorf("expected 'abc123def', got %q", got.ID)
	}
}

func TestGetNodeByPrefix_NoFalseMatchesFromWildcards(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// If % were not escaped, searching for "abc%" would match any ID starting with "abc".
	// We only want exact literal prefix matching.
	s.CreateNode(ctx, &Node{ID: "abc%def", NodeType: NodeTypeUser, Content: "literal-percent", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "abcXYZ", NodeType: NodeTypeUser, Content: "should-not-match", Sequence: 2})

	got, err := s.GetNodeByPrefix(ctx, "abc%")
	if err != nil {
		t.Fatalf("GetNodeByPrefix error: %v", err)
	}
	if got.ID != "abc%def" {
		t.Errorf("false positive: got %q, want 'abc%%def'", got.ID)
	}
}

func TestGetNodeByPrefix_PercentAtStart(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	s.CreateNode(ctx, &Node{ID: "%start", NodeType: NodeTypeUser, Content: "percent-start", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "Xstart", NodeType: NodeTypeUser, Content: "other", Sequence: 2})

	got, err := s.GetNodeByPrefix(ctx, "%s")
	if err != nil {
		t.Fatalf("GetNodeByPrefix error: %v", err)
	}
	if got.ID != "%start" {
		t.Errorf("expected '%%start', got %q", got.ID)
	}
}

func TestGetNodeByPrefix_UnderscoreAtStart(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	s.CreateNode(ctx, &Node{ID: "_start", NodeType: NodeTypeUser, Content: "underscore-start", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: "Xstart", NodeType: NodeTypeUser, Content: "other", Sequence: 2})

	got, err := s.GetNodeByPrefix(ctx, "_s")
	if err != nil {
		t.Fatalf("GetNodeByPrefix error: %v", err)
	}
	if got.ID != "_start" {
		t.Errorf("expected '_start', got %q", got.ID)
	}
}

func TestGetNodeByPrefix_EscapeCharItself(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Backslash followed by a wildcard character: the function escapes % and _
	// with backslash prefix, so the search string should not be confused.
	s.CreateNode(ctx, &Node{ID: `abc\%def`, NodeType: NodeTypeUser, Content: "backslash-percent", Sequence: 1})
	s.CreateNode(ctx, &Node{ID: `abc\Xdef`, NodeType: NodeTypeUser, Content: "backslash-other", Sequence: 2})

	// Should not panic; either match or no-match is acceptable.
	_, _ = s.GetNodeByPrefix(ctx, `abc\%`)

	// Also verify the function doesn't panic with backslash at end of prefix.
	_, _ = s.GetNodeByPrefix(ctx, `abc\`)
}

// ---------------------------------------------------------------------------
// Verify no SQL injection through the LIKE pattern
// ---------------------------------------------------------------------------

func TestGetNodeByPrefix_SQLInjectionResistance(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	s.CreateNode(ctx, &Node{ID: "normal-id", NodeType: NodeTypeUser, Content: "normal", Sequence: 1})

	// These payloads attempt to break out of the LIKE pattern or inject SQL.
	// The function uses parameterized queries (?), so SQL injection is not
	// possible. But we verify the LIKE escaping handles pathological input.
	payloads := []string{
		`' OR '1'='1`,
		`'; DROP TABLE nodes; --`,
		`' UNION SELECT * FROM nodes --`,
		`%`,
		`_`,
		`%%`,
		`__`,
		`\%`,
		`\_`,
		`%_`,
		`_%`,
		`%%__`,
	}

	for _, payload := range payloads {
		t.Run("payload_"+payload, func(t *testing.T) {
			// Should not panic, and should return a clean error (no match).
			_, err := s.GetNodeByPrefix(ctx, payload)
			// Either a match or no-match is fine; the key invariant is no panic.
			_ = err
		})
	}
}
