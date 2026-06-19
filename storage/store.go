package storage

import "context"

// Store captures the persistence operations used by the DAG wrapper and API
// layers so tests can swap implementations without changing call sites.
type Store interface {
	CreateNode(ctx context.Context, node *Node) error
	GetNode(ctx context.Context, id string) (*Node, error)
	GetNodeByPrefix(ctx context.Context, prefix string) (*Node, error)
	GetNodeChildren(ctx context.Context, id string) ([]*Node, error)
	GetSubtree(ctx context.Context, id string) ([]*Node, error)
	GetAncestors(ctx context.Context, id string) ([]*Node, error)
	ListRootNodes(ctx context.Context) ([]*Node, error)
	UpdateNode(ctx context.Context, node *Node) error
	DeleteNode(ctx context.Context, id string) error
	CreateAlias(ctx context.Context, alias, nodeID string) error
	DeleteAlias(ctx context.Context, alias string) error
	GetNodeByAlias(ctx context.Context, alias string) (*Node, error)
	ListAliases(ctx context.Context, nodeID string) ([]Alias, error)
	IndexToolIDs(ctx context.Context, nodeID string, toolIDs []string, role string) error
	// GetOrphanedToolUses returns tool_use IDs on ancestorIDs with no matching
	// tool_result on the same path. Map keys are node IDs.
	GetOrphanedToolUses(ctx context.Context, ancestorIDs []string) (map[string][]string, error)
	Close() error
}
