package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func Open(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("storage: open %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func (s *SQLiteStore) migrate() error {
	_, err := s.db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS nodes (
			id TEXT PRIMARY KEY,
			parent_id TEXT REFERENCES nodes(id) ON DELETE CASCADE,
			root_id TEXT,
			sequence INTEGER NOT NULL DEFAULT 0,
			node_type TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			provider TEXT DEFAULT '',
			model TEXT DEFAULT '',
			tokens_in INTEGER DEFAULT 0,
			tokens_out INTEGER DEFAULT 0,
			tokens_cache_read INTEGER DEFAULT 0,
			tokens_cache_creation INTEGER DEFAULT 0,
			tokens_reasoning INTEGER DEFAULT 0,
			latency_ms INTEGER DEFAULT 0,
			stop_reason TEXT DEFAULT '',
			output_group_id TEXT DEFAULT '',
			status TEXT DEFAULT '',
			title TEXT DEFAULT '',
			system_prompt TEXT DEFAULT '',
			metadata TEXT DEFAULT '',
			created_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_nodes_parent ON nodes(parent_id);
		CREATE INDEX IF NOT EXISTS idx_nodes_root ON nodes(root_id);
		CREATE INDEX IF NOT EXISTS idx_nodes_roots ON nodes(root_id) WHERE parent_id IS NULL;
		CREATE INDEX IF NOT EXISTS idx_nodes_output_group ON nodes(output_group_id) WHERE output_group_id != '';

		CREATE TABLE IF NOT EXISTS node_aliases (
			alias TEXT PRIMARY KEY,
			node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS node_tool_ids (
			node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
			tool_id TEXT NOT NULL,
			role TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tool_ids_node ON node_tool_ids(node_id);
		CREATE INDEX IF NOT EXISTS idx_tool_ids_role ON node_tool_ids(role);
	`)
	return err
}

func (s *SQLiteStore) CreateNode(ctx context.Context, node *Node) error {
	if node.CreatedAt.IsZero() {
		node.CreatedAt = time.Now()
	}
	meta := ""
	if node.Metadata != nil {
		meta = string(node.Metadata)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO nodes (id, parent_id, root_id, sequence, node_type, content, provider, model, tokens_in, tokens_out, tokens_cache_read, tokens_cache_creation, tokens_reasoning, latency_ms, stop_reason, output_group_id, status, title, system_prompt, metadata, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		node.ID, nilIfEmpty(node.ParentID), nilIfEmpty(node.RootID), node.Sequence, node.NodeType, node.Content,
		node.Provider, node.Model, node.TokensIn, node.TokensOut, node.TokensCacheRead, node.TokensCacheCreation,
		node.TokensReasoning, node.LatencyMs, node.StopReason, node.OutputGroupID, node.Status,
		node.Title, node.SystemPrompt, meta, node.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func (s *SQLiteStore) GetNode(ctx context.Context, id string) (*Node, error) {
	return s.scanNode(s.db.QueryRowContext(ctx, `SELECT id, parent_id, root_id, sequence, node_type, content, provider, model, tokens_in, tokens_out, tokens_cache_read, tokens_cache_creation, tokens_reasoning, latency_ms, stop_reason, output_group_id, status, title, system_prompt, metadata, created_at FROM nodes WHERE id = ?`, id))
}

func (s *SQLiteStore) GetNodeByPrefix(ctx context.Context, prefix string) (*Node, error) {
	// Escape SQL LIKE wildcards in the prefix to prevent false matches.
	escaped := strings.NewReplacer("%", "\\%", "_", "\\_").Replace(prefix)
	return s.scanNode(s.db.QueryRowContext(ctx, `SELECT id, parent_id, root_id, sequence, node_type, content, provider, model, tokens_in, tokens_out, tokens_cache_read, tokens_cache_creation, tokens_reasoning, latency_ms, stop_reason, output_group_id, status, title, system_prompt, metadata, created_at FROM nodes WHERE id LIKE ? ESCAPE '\' LIMIT 1`, escaped+"%"))
}

func (s *SQLiteStore) GetNodeChildren(ctx context.Context, id string) ([]*Node, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, parent_id, root_id, sequence, node_type, content, provider, model, tokens_in, tokens_out, tokens_cache_read, tokens_cache_creation, tokens_reasoning, latency_ms, stop_reason, output_group_id, status, title, system_prompt, metadata, created_at FROM nodes WHERE parent_id = ? ORDER BY sequence`, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanNodes(rows)
}

func (s *SQLiteStore) GetSubtree(ctx context.Context, id string) ([]*Node, error) {
	rows, err := s.db.QueryContext(ctx, `
		WITH RECURSIVE tree AS (
			SELECT id, parent_id, root_id, sequence, node_type, content, provider, model, tokens_in, tokens_out, tokens_cache_read, tokens_cache_creation, tokens_reasoning, latency_ms, stop_reason, output_group_id, status, title, system_prompt, metadata, created_at FROM nodes WHERE id = ?
			UNION ALL
			SELECT n.id, n.parent_id, n.root_id, n.sequence, n.node_type, n.content, n.provider, n.model, n.tokens_in, n.tokens_out, n.tokens_cache_read, n.tokens_cache_creation, n.tokens_reasoning, n.latency_ms, n.stop_reason, n.output_group_id, n.status, n.title, n.system_prompt, n.metadata, n.created_at FROM nodes n JOIN tree t ON n.parent_id = t.id
		) SELECT * FROM tree ORDER BY sequence`, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanNodes(rows)
}

func (s *SQLiteStore) GetAncestors(ctx context.Context, id string) ([]*Node, error) {
	rows, err := s.db.QueryContext(ctx, `
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_id, root_id, sequence, node_type, content, provider, model, tokens_in, tokens_out, tokens_cache_read, tokens_cache_creation, tokens_reasoning, latency_ms, stop_reason, output_group_id, status, title, system_prompt, metadata, created_at FROM nodes WHERE id = ?
			UNION ALL
			SELECT n.id, n.parent_id, n.root_id, n.sequence, n.node_type, n.content, n.provider, n.model, n.tokens_in, n.tokens_out, n.tokens_cache_read, n.tokens_cache_creation, n.tokens_reasoning, n.latency_ms, n.stop_reason, n.output_group_id, n.status, n.title, n.system_prompt, n.metadata, n.created_at FROM nodes n JOIN ancestors a ON n.id = a.parent_id
		) SELECT * FROM ancestors ORDER BY sequence`, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanNodes(rows)
}

func (s *SQLiteStore) ListRootNodes(ctx context.Context) ([]*Node, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, parent_id, root_id, sequence, node_type, content, provider, model, tokens_in, tokens_out, tokens_cache_read, tokens_cache_creation, tokens_reasoning, latency_ms, stop_reason, output_group_id, status, title, system_prompt, metadata, created_at FROM nodes WHERE parent_id IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanNodes(rows)
}

func (s *SQLiteStore) UpdateNode(ctx context.Context, node *Node) error {
	meta := ""
	if node.Metadata != nil {
		meta = string(node.Metadata)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE nodes SET content=?, provider=?, model=?, tokens_in=?, tokens_out=?, tokens_cache_read=?, tokens_cache_creation=?, tokens_reasoning=?, latency_ms=?, stop_reason=?, output_group_id=?, status=?, title=?, system_prompt=?, metadata=? WHERE id=?`,
		node.Content, node.Provider, node.Model, node.TokensIn, node.TokensOut, node.TokensCacheRead, node.TokensCacheCreation,
		node.TokensReasoning, node.LatencyMs, node.StopReason, node.OutputGroupID, node.Status,
		node.Title, node.SystemPrompt, meta, node.ID)
	return err
}

func (s *SQLiteStore) DeleteNode(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM nodes WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) CreateAlias(ctx context.Context, alias, nodeID string) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO node_aliases (alias, node_id) VALUES (?, ?)`, alias, nodeID)
	return err
}

func (s *SQLiteStore) DeleteAlias(ctx context.Context, alias string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM node_aliases WHERE alias = ?`, alias)
	return err
}

func (s *SQLiteStore) GetNodeByAlias(ctx context.Context, alias string) (*Node, error) {
	var nodeID string
	if err := s.db.QueryRowContext(ctx, `SELECT node_id FROM node_aliases WHERE alias = ?`, alias).Scan(&nodeID); err != nil {
		return nil, err
	}
	return s.GetNode(ctx, nodeID)
}

func (s *SQLiteStore) ListAliases(ctx context.Context, nodeID string) ([]Alias, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT alias, node_id FROM node_aliases WHERE node_id = ?`, nodeID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var aliases []Alias
	for rows.Next() {
		var a Alias
		if err := rows.Scan(&a.Alias, &a.NodeID); err != nil {
			return nil, err
		}
		aliases = append(aliases, a)
	}
	return aliases, rows.Err()
}

func (s *SQLiteStore) IndexToolIDs(ctx context.Context, nodeID string, toolIDs []string, role string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO node_tool_ids (node_id, tool_id, role) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()
	for _, tid := range toolIDs {
		if _, err := stmt.ExecContext(ctx, nodeID, tid, role); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) GetOrphanedToolUses(ctx context.Context, ancestorIDs []string) (map[string][]string, error) {
	if len(ancestorIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ancestorIDs))
	args := make([]any, 0, len(ancestorIDs)*2)
	for i, id := range ancestorIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	inClause := strings.Join(placeholders, ",")
	for _, id := range ancestorIDs {
		args = append(args, id)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT nti.node_id, nti.tool_id
		FROM node_tool_ids nti
		WHERE nti.node_id IN (`+inClause+`) AND nti.role = 'use'
		AND nti.tool_id NOT IN (
			SELECT tool_id FROM node_tool_ids
			WHERE node_id IN (`+inClause+`) AND role = 'result'
		)`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string][]string)
	for rows.Next() {
		var nodeID, toolID string
		if err := rows.Scan(&nodeID, &toolID); err != nil {
			return nil, err
		}
		result[nodeID] = append(result[nodeID], toolID)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) scanNode(row *sql.Row) (*Node, error) {
	var n Node
	var parentID, rootID sql.NullString
	var meta, createdAt string
	err := row.Scan(&n.ID, &parentID, &rootID, &n.Sequence, &n.NodeType, &n.Content,
		&n.Provider, &n.Model, &n.TokensIn, &n.TokensOut, &n.TokensCacheRead, &n.TokensCacheCreation,
		&n.TokensReasoning, &n.LatencyMs, &n.StopReason, &n.OutputGroupID, &n.Status,
		&n.Title, &n.SystemPrompt, &meta, &createdAt)
	if err != nil {
		return nil, err
	}
	n.ParentID = parentID.String
	n.RootID = rootID.String
	if meta != "" {
		n.Metadata = json.RawMessage(meta)
	}
	n.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return &n, nil
}

func (s *SQLiteStore) scanNodes(rows *sql.Rows) ([]*Node, error) {
	var nodes []*Node
	for rows.Next() {
		var n Node
		var parentID, rootID sql.NullString
		var meta, createdAt string
		err := rows.Scan(&n.ID, &parentID, &rootID, &n.Sequence, &n.NodeType, &n.Content,
			&n.Provider, &n.Model, &n.TokensIn, &n.TokensOut, &n.TokensCacheRead, &n.TokensCacheCreation,
			&n.TokensReasoning, &n.LatencyMs, &n.StopReason, &n.OutputGroupID, &n.Status,
			&n.Title, &n.SystemPrompt, &meta, &createdAt)
		if err != nil {
			return nil, err
		}
		n.ParentID = parentID.String
		n.RootID = rootID.String
		if meta != "" {
			n.Metadata = json.RawMessage(meta)
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		nodes = append(nodes, &n)
	}
	return nodes, rows.Err()
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
