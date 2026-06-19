package storage

import (
	"encoding/json"
	"time"
)

// NodeType identifies the role of a persisted conversation node.
type NodeType string

const (
	NodeTypeUser       NodeType = "user"
	NodeTypeAssistant  NodeType = "assistant"
	NodeTypeSystem     NodeType = "system"
	NodeTypeToolCall   NodeType = "tool_call"
	NodeTypeToolResult NodeType = "tool_result"
)

// Node is a single persisted entry in the conversation DAG.
type Node struct {
	ID                  string          `json:"id"`
	ParentID            string          `json:"parent_id,omitempty"`
	RootID              string          `json:"root_id,omitempty"`
	Sequence            int             `json:"sequence"`
	NodeType            NodeType        `json:"node_type"`
	Content             string          `json:"content"`
	Provider            string          `json:"provider,omitempty"`
	Model               string          `json:"model,omitempty"`
	TokensIn            int             `json:"tokens_in,omitempty"`
	TokensOut           int             `json:"tokens_out,omitempty"`
	TokensCacheRead     int             `json:"tokens_cache_read,omitempty"`
	TokensCacheCreation int             `json:"tokens_cache_creation,omitempty"`
	TokensReasoning     int             `json:"tokens_reasoning,omitempty"`
	LatencyMs           int             `json:"latency_ms,omitempty"`
	StopReason          string          `json:"stop_reason,omitempty"`
	OutputGroupID       string          `json:"output_group_id,omitempty"`
	Status              string          `json:"status,omitempty"`
	Title               string          `json:"title,omitempty"`
	SystemPrompt        string          `json:"system_prompt,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	Metadata            json.RawMessage `json:"metadata,omitempty"`
}

// Alias maps a stable name to a node ID.
type Alias struct {
	Alias  string `json:"alias"`
	NodeID string `json:"node_id"`
}
