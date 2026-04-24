# Proto

This directory stores gRPC contracts between `backend` and `core-agent`.

## Contract

- [`agent/v1/agent.proto`](./agent/v1/agent.proto)

## Generate Go Stubs

Prerequisites:
- `protoc`
- `protoc-gen-go`
- `protoc-gen-go-grpc`

Recommended command from the repository root:

```bash
make proto-go
```

This generates the checked-in Go stubs consumed by the backend under:

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

## Generate Rust Stubs

Recommended approach (tonic + tonic-build via `build.rs`):

1. Add dependencies in `core-agent/Cargo.toml`:
   - `tonic`
   - `prost`
   - build dependency `tonic-build`
2. Add a `build.rs` invoking `tonic_build::configure().compile(...)`.
3. Run:

```bash
cargo build
```
