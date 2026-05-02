package client

// EmbeddingParams holds asymmetric params for indexing vs query.
type EmbeddingParams struct {
	Indexing map[string]string `json:"indexing,omitempty"`
	Query    map[string]string `json:"query,omitempty"`
}

// EmbeddingRequest represents an embedding API call.
type EmbeddingRequest struct {
	Model  string            `json:"model"`
	Input  []string          `json:"input"`
	Params map[string]string `json:"params,omitempty"` // indexing or query params
}

// EmbeddingResponse holds embedding results.
type EmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
	Model      string      `json:"model"`
	Usage      *EyrieUsage `json:"usage,omitempty"`
}
