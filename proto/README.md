# Proto

This directory stores gRPC contracts between `backend` and `core-agent`.

## Contract

- [`agent/v1/agent.proto`](./agent/v1/agent.proto)

## Generate Go Stubs

Prerequisites:
- `protoc`
- `protoc-gen-go`
- `protoc-gen-go-grpc`

Command:

```bash
protoc \
  --proto_path=. \
  --go_out=. \
  --go-grpc_out=. \
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
