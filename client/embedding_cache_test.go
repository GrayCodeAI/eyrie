package client

import (
	"context"
	"strings"
	"testing"
)

// stubEmbedder returns a deterministic vector chosen by keyword in the input,
// so the test can control which requests are "similar".
type stubEmbedder struct{}

func (stubEmbedder) CreateEmbedding(_ context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	in := ""
	if len(req.Input) > 0 {
		in = req.Input[0]
	}
	var vec []float32
	switch {
	case strings.Contains(in, "weather"):
		vec = []float32{1, 0, 0}
	case strings.Contains(in, "forecast"):
		vec = []float32{0.99, 0.1, 0} // ~0.995 cosine vs weather → hit
	case strings.Contains(in, "database"):
		vec = []float32{0, 1, 0} // orthogonal → miss
	default:
		vec = []float32{0, 0, 1}
	}
	return &EmbeddingResponse{Embeddings: [][]float32{vec}}, nil
}

func userMsg(s string) []EyrieMessage { return []EyrieMessage{{Role: "user", Content: s}} }

func TestEmbeddingCache_HitOnSimilarPrompt(t *testing.T) {
	mock := NewMockProvider(MockModeEcho)
	sp := NewEmbeddingCachedProvider(mock, stubEmbedder{}, DefaultSemanticCacheConfig())
	ctx := context.Background()

	first, err := sp.Chat(ctx, userMsg("what is the weather today"), ChatOptions{})
	if err != nil {
		t.Fatalf("first chat: %v", err)
	}

	// A differently-worded but semantically-similar prompt should return the
	// cached response, not a fresh echo.
	second, err := sp.Chat(ctx, userMsg("give me the weather please"), ChatOptions{})
	if err != nil {
		t.Fatalf("second chat: %v", err)
	}
	if second.Content != first.Content {
		t.Errorf("expected cache hit returning first response %q, got %q", first.Content, second.Content)
	}
	if mock.CallCount() != 1 {
		t.Errorf("expected inner provider called once, got %d", mock.CallCount())
	}
	if st := sp.Stats(); st.Hits != 1 || st.Misses != 1 {
		t.Errorf("unexpected stats: %+v", st)
	}
}

func TestEmbeddingCache_MissOnDissimilarPrompt(t *testing.T) {
	mock := NewMockProvider(MockModeEcho)
	sp := NewEmbeddingCachedProvider(mock, stubEmbedder{}, DefaultSemanticCacheConfig())
	ctx := context.Background()

	if _, err := sp.Chat(ctx, userMsg("weather forecast lookup"), ChatOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := sp.Chat(ctx, userMsg("database schema migration"), ChatOptions{}); err != nil {
		t.Fatal(err)
	}
	if mock.CallCount() != 2 {
		t.Errorf("expected 2 inner calls for dissimilar prompts, got %d", mock.CallCount())
	}
}

func TestEmbeddingCache_SkipsHighTemperature(t *testing.T) {
	mock := NewMockProvider(MockModeEcho)
	sp := NewEmbeddingCachedProvider(mock, stubEmbedder{}, DefaultSemanticCacheConfig())
	ctx := context.Background()
	temp := 0.9

	_, _ = sp.Chat(ctx, userMsg("weather today"), ChatOptions{Temperature: &temp})
	_, _ = sp.Chat(ctx, userMsg("weather today"), ChatOptions{Temperature: &temp})
	if mock.CallCount() != 2 {
		t.Errorf("high-temperature requests should bypass cache; got %d calls", mock.CallCount())
	}
}

func TestEmbeddingCache_DegradesOnEmbedError(t *testing.T) {
	mock := NewMockProvider(MockModeEcho)
	sp := NewEmbeddingCachedProvider(mock, errEmbedder{}, DefaultSemanticCacheConfig())
	if _, err := sp.Chat(context.Background(), userMsg("weather"), ChatOptions{}); err != nil {
		t.Fatalf("expected graceful degradation, got error: %v", err)
	}
	if mock.CallCount() != 1 {
		t.Errorf("expected inner call on embed failure, got %d", mock.CallCount())
	}
}

type errEmbedder struct{}

func (errEmbedder) CreateEmbedding(_ context.Context, _ EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, errEmptyEmbedding
}

func TestCosineSimilarity(t *testing.T) {
	if got := cosineSimilarity([]float32{1, 0}, []float32{1, 0}); got < 0.999 {
		t.Errorf("identical vectors should be ~1, got %f", got)
	}
	if got := cosineSimilarity([]float32{1, 0}, []float32{0, 1}); got != 0 {
		t.Errorf("orthogonal vectors should be 0, got %f", got)
	}
	if got := cosineSimilarity([]float32{1, 0}, []float32{1}); got != 0 {
		t.Errorf("mismatched lengths should be 0, got %f", got)
	}
}
