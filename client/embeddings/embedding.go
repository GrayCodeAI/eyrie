package client

import (
	"context"
	"fmt"
)

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

// CreateEmbedding sends an embedding request to the specified (or default) provider.
func (c *EyrieClient) CreateEmbedding(ctx context.Context, req EmbeddingRequest, provider string) (*EmbeddingResponse, error) {
	if provider == "" {
		provider = c.defaultProvider
	}
	p, err := c.getOrCreateProvider(provider)
	if err != nil {
		return nil, err
	}
	embedder, ok := p.(Embedder)
	if !ok {
		return nil, fmt.Errorf("eyrie: provider %s does not support embeddings", provider)
	}
	return embedder.CreateEmbedding(ctx, req)
}
