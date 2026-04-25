# Proto

This directory stores gRPC contracts between `backend` and `core-agent`.

## Contract

- [`agent/v1/agent.proto`](./agent/v1/agent.proto)

## Generate Go Bindings

Prerequisites:
- `protoc`
- `protoc-gen-go`
- `protoc-gen-go-grpc`

Recommended command from the repository root:

```bash
make proto-go
```

This generates the checked-in Go protobuf/gRPC bindings consumed by the backend under:

- `backend/internal/grpcclient/pb/proto/agent/v1/agent.pb.go`
- `backend/internal/grpcclient/pb/proto/agent/v1/agent_grpc.pb.go`

Equivalent raw command:

```bash
protoc \
  --proto_path=. \
  --go_out=paths=source_relative:backend/internal/grpcclient/pb \
  --go-grpc_out=paths=source_relative:backend/internal/grpcclient/pb \
  proto/agent/v1/agent.proto
```

## Generate Rust Bindings

Rust protobuf/gRPC bindings are generated during `core-agent` builds via `build.rs` (`tonic-build` + vendored `protoc`).

Run:

```bash
cd core-agent && cargo build
```
