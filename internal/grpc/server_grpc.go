//go:build grpc

// This file is the real gRPC server wiring. It is excluded from normal builds
// via the "grpc" build tag because eyrie does not yet depend on
// google.golang.org/grpc. It is intentionally a documented stub: it compiles
// under `go build -tags grpc ./...` without pulling in grpc, and marks exactly
// where the generated code and server registration belong.
//
// To activate (see README.md for full steps):
//  1. go get google.golang.org/grpc google.golang.org/protobuf
//  2. generate eyriev1 stubs from proto/eyrie/v1/chat.proto
//  3. replace the body of Serve below with a real grpc.Server that registers
//     a ChatServiceServer adapting conversation.Engine, and drop this note.

package grpc

import (
	"errors"
	"net"
)

// errGRPCNotWired signals that the gRPC dependency/codegen has not been added.
var errGRPCNotWired = errors.New("eyrie/grpc: gRPC server not wired up (run codegen; see internal/grpc/README.md)")

// Serve will listen on lis and serve the ChatService over gRPC once the
// dependency and generated stubs are in place. Today it is a stub.
//
// TODO(grpc): construct grpc.NewServer(), register the generated
// ChatServiceServer backed by svc, and call grpcServer.Serve(lis).
func Serve(_ net.Listener, _ ChatService) error {
	return errGRPCNotWired
}
