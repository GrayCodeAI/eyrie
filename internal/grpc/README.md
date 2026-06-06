# internal/grpc — gRPC server skeleton (design note)

This package is a **skeleton**. eyrie does not currently depend on
`google.golang.org/grpc` (verify with `go.mod`), and per repo policy we do
**not** add that dependency speculatively. So this package ships:

- `grpc.go` — a dependency-free `ChatService` interface plus a no-op default
  implementation. This always compiles and lets the rest of the codebase
  reference the service contract today.
- `server_grpc.go` — the real gRPC wiring, guarded by the `grpc` build tag so
  it is excluded from normal builds until the dependency and generated code
  exist. It is currently a documented stub (no grpc imports) describing exactly
  what to generate and wire.

## What to generate when gRPC is adopted

1. Add the dependencies:

   ```sh
   go get google.golang.org/grpc google.golang.org/protobuf
   ```

2. Define the service in `proto/eyrie/v1/chat.proto`:

   ```proto
   syntax = "proto3";
   package eyrie.v1;
   option go_package = "github.com/GrayCodeAI/eyrie/internal/grpc/eyriev1";

   service ChatService {
     // Unary chat: one request, one response.
     rpc Chat(ChatRequest) returns (ChatResponse);
   }

   message ChatRequest {
     string model = 1;
     string system_prompt = 2;
     string message = 3;
     int32  max_tokens = 4;
   }

   message ChatResponse {
     string content = 1;
     string node_id = 2;
     string finish_reason = 3;
   }
   ```

3. Generate stubs with `protoc` (or `buf`):

   ```sh
   protoc --go_out=. --go-grpc_out=. proto/eyrie/v1/chat.proto
   ```

4. Implement the generated `ChatServiceServer` by adapting the existing
   `conversation.Engine` (see `internal/api/server.go` `handlePrompt` for the
   HTTP equivalent), then register it on a `grpc.Server` inside
   `server_grpc.go` (drop the stub body, remove the documentation comment).

5. Build/run with the tag: `go build -tags grpc ./...`.

Until then, importing this package yields the no-op `ChatService` in
`grpc.go`, which keeps everything compiling with zero new dependencies.
