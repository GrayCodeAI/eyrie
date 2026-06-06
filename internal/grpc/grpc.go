// Package grpc holds a dependency-free skeleton for an eyrie gRPC API.
//
// eyrie does not currently import google.golang.org/grpc, and per repo policy
// that dependency is not added speculatively. This file therefore defines only
// the service contract and a no-op default implementation so the rest of the
// codebase can reference the gRPC surface today. The real server wiring lives
// in server_grpc.go behind the "grpc" build tag. See README.md for the design
// note and codegen steps.
package grpc

import "context"

// ChatRequest is the unary Chat request payload. It mirrors the HTTP
// /prompt request fields so a gRPC implementation can reuse the conversation
// engine without translation.
type ChatRequest struct {
	Model        string
	SystemPrompt string
	Message      string
	MaxTokens    int
}

// ChatResponse is the unary Chat response payload.
type ChatResponse struct {
	Content      string
	NodeID       string
	FinishReason string
}

// ChatService is the eyrie gRPC service contract: a single unary Chat RPC.
// A concrete implementation will adapt conversation.Engine; see README.md.
type ChatService interface {
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}

// noopChatService is the default ChatService. It returns ErrUnimplemented so
// callers get a clear signal that the gRPC backend has not been wired up.
type noopChatService struct{}

// ErrUnimplemented is returned by the default ChatService until a real
// gRPC-backed implementation is provided.
//
// TODO(grpc): replace noopChatService with an engine-backed implementation
// once google.golang.org/grpc and the generated stubs are available (see
// README.md), and register it via server_grpc.go (build tag "grpc").
var ErrUnimplemented = errUnimplemented{}

type errUnimplemented struct{}

func (errUnimplemented) Error() string { return "eyrie/grpc: ChatService not implemented" }

func (noopChatService) Chat(_ context.Context, _ *ChatRequest) (*ChatResponse, error) {
	return nil, ErrUnimplemented
}

// NewChatService returns the default (no-op) ChatService. It exists so callers
// have a stable constructor; once a real backend exists this will return the
// engine-backed implementation instead.
func NewChatService() ChatService {
	return noopChatService{}
}
