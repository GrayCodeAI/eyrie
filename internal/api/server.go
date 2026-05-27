package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/GrayCodeAI/eyrie/client"
	"github.com/GrayCodeAI/eyrie/conversation"
	eyrie "github.com/GrayCodeAI/eyrie/internal/health"
	"github.com/GrayCodeAI/eyrie/storage"
)

const maxRequestBodyBytes = 1 << 20

type Server struct {
	engine        *conversation.Engine
	store         storage.Store
	analytics     storage.AnalyticsStore
	healthChecker *eyrie.HealthChecker
	apiKey        string
	mux           *http.ServeMux
	bgCtx         context.Context
}

type Config struct {
	Store         storage.Store
	Analytics     storage.AnalyticsStore // optional: enables /api/usage, /api/costs
	Provider      client.Provider
	HealthChecker *eyrie.HealthChecker // optional: enables /api/health/providers
	APIKey        string
	Port          int
}

func NewServer(cfg Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	_ = cancel // cancelled on server shutdown if needed
	s := &Server{
		engine:        conversation.New(cfg.Store, cfg.Provider),
		store:         cfg.Store,
		analytics:     cfg.Analytics,
		healthChecker: cfg.HealthChecker,
		apiKey:        cfg.APIKey,
		mux:           http.NewServeMux(),
		bgCtx:         ctx,
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) ListenAndServe(addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      10 * time.Minute, // long for streaming LLM responses
		IdleTimeout:       120 * time.Second,
	}
	return srv.ListenAndServe()
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("POST /prompt", s.auth(s.handlePrompt))
	s.mux.HandleFunc("POST /nodes/{id}/prompt", s.auth(s.handlePromptFrom))
	s.mux.HandleFunc("GET /nodes", s.auth(s.handleListNodes))
	s.mux.HandleFunc("GET /nodes/{id}", s.auth(s.handleGetNode))
	s.mux.HandleFunc("GET /nodes/{id}/tree", s.auth(s.handleGetTree))
	s.mux.HandleFunc("DELETE /nodes/{id}", s.auth(s.handleDeleteNode))
	s.mux.HandleFunc("PUT /nodes/{id}/aliases/{alias}", s.auth(s.handleCreateAlias))
	s.mux.HandleFunc("DELETE /aliases/{alias}", s.auth(s.handleDeleteAlias))

	// Analytics and health dashboard endpoints.
	s.mux.HandleFunc("GET /api/usage", s.auth(s.handleUsageAnalytics))
	s.mux.HandleFunc("GET /api/costs", s.auth(s.handleCostSummary))
	s.mux.HandleFunc("GET /api/health/providers", s.auth(s.handleProviderHealth))
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.apiKey == "" {
			next(w, r)
			return
		}
		token := r.Header.Get("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")
		if token == "" {
			token = r.Header.Get("X-API-Key")
		}
		if !constantTimeEqual(token, s.apiKey) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r)
	}
}

func constantTimeEqual(a, b string) bool {
	// Pad the shorter value so comparison time does not leak token length.
	if len(a) < len(b) {
		a += strings.Repeat("\x00", len(b)-len(a))
	} else if len(b) < len(a) {
		b += strings.Repeat("\x00", len(a)-len(b))
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return false
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body must contain a single JSON object"})
		return false
	}
	return true
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type promptRequest struct {
	Message      string             `json:"message"`
	Model        string             `json:"model,omitempty"`
	SystemPrompt string             `json:"system_prompt,omitempty"`
	MaxTokens    int                `json:"max_tokens,omitempty"`
	Stream       bool               `json:"stream,omitempty"`
	Tools        []client.EyrieTool `json:"tools,omitempty"`
}

func (s *Server) handlePrompt(w http.ResponseWriter, r *http.Request) {
	var req promptRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}

	opts := conversation.PromptOpts{
		Model:        req.Model,
		SystemPrompt: req.SystemPrompt,
		MaxTokens:    req.MaxTokens,
		Tools:        req.Tools,
	}

	if req.Stream {
		s.streamResponse(w, r.Context(), func(ctx context.Context) (<-chan conversation.Event, error) {
			return s.engine.Prompt(ctx, req.Message, opts)
		})
		return
	}

	events, err := s.engine.Prompt(r.Context(), req.Message, opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.collectAndRespond(w, events)
}

func (s *Server) handlePromptFrom(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	var req promptRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}

	opts := conversation.PromptOpts{
		Model:        req.Model,
		SystemPrompt: req.SystemPrompt,
		MaxTokens:    req.MaxTokens,
		Tools:        req.Tools,
	}

	if req.Stream {
		s.streamResponse(w, r.Context(), func(ctx context.Context) (<-chan conversation.Event, error) {
			return s.engine.PromptFrom(ctx, nodeID, req.Message, opts)
		})
		return
	}

	events, err := s.engine.PromptFrom(r.Context(), nodeID, req.Message, opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.collectAndRespond(w, events)
}

func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.store.ListRootNodes(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, nodes)
}

func (s *Server) handleGetNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	node, err := s.engine.ResolveNode(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "node not found"})
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (s *Server) handleGetTree(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	nodes, err := s.store.GetSubtree(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, nodes)
}

func (s *Server) handleDeleteNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteNode(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleCreateAlias(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	alias := r.PathValue("alias")
	if err := s.store.CreateAlias(r.Context(), alias, nodeID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"alias": alias, "node_id": nodeID})
}

func (s *Server) handleDeleteAlias(w http.ResponseWriter, r *http.Request) {
	alias := r.PathValue("alias")
	if err := s.store.DeleteAlias(r.Context(), alias); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) streamResponse(w http.ResponseWriter, ctx context.Context, start func(context.Context) (<-chan conversation.Event, error)) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	events, err := start(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for ev := range events {
		data, _ := json.Marshal(ev)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

func (s *Server) collectAndRespond(w http.ResponseWriter, events <-chan conversation.Event) {
	var content string
	var nodeID string
	var errMsg string
	for ev := range events {
		switch ev.Type {
		case conversation.EventDelta:
			content += ev.Content
		case conversation.EventDone:
			nodeID = ev.NodeID
		case conversation.EventError:
			errMsg = ev.Error
		}
	}
	if errMsg != "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": errMsg})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"content": content,
		"node_id": nodeID,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
