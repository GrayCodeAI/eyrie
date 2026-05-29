package types

import (
	"regexp"
)

// --- Branded string types ---

type (
	SessionId string
	AgentId   string
)

var agentIdPattern = regexp.MustCompile(`^a(?:.+-)?[0-9a-f]{16}$`)

func AsSessionId(s string) SessionId { return SessionId(s) }
func AsAgentId(s string) AgentId     { return AgentId(s) }

func ToAgentId(s string) (*AgentId, error) {
	if !agentIdPattern.MatchString(s) {
		return nil, nil
	}
	id := AgentId(s)
	return &id, nil
}

// --- Connector types ---

type ConnectorTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ConnectorTextDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func IsConnectorTextBlock(m map[string]interface{}) bool {
	t, ok := m["type"].(string)
	return ok && t == "connector_text"
}

// --- Usage types ---

type ServerToolUse struct {
	WebSearchRequests int `json:"web_search_requests"`
	WebFetchRequests  int `json:"web_fetch_requests"`
}

type CacheCreation struct {
	Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
	Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
}

type NonNullableUsage struct {
	InputTokens              int           `json:"input_tokens"`
	CacheCreationInputTokens int           `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int           `json:"cache_read_input_tokens"`
	OutputTokens             int           `json:"output_tokens"`
	ServerToolUse            ServerToolUse `json:"server_tool_use"`
	ServiceTier              string        `json:"service_tier"`
	CacheCreation            CacheCreation `json:"cache_creation"`
	InferenceGeo             string        `json:"inference_geo"`
	Iterations               []interface{} `json:"iterations"`
	Speed                    string        `json:"speed"`
}

func EmptyUsage() NonNullableUsage {
	return NonNullableUsage{Iterations: []interface{}{}}
}
