# Core Agent

Rust gRPC core-agent baseline using:
- `tonic` + `tokio`
- `tracing`
- `sysinfo`

## Run

`cargo run`

Environment:
- `CORE_AGENT_HOST` (default `0.0.0.0`)
- `CORE_AGENT_PORT` (default `50051`)
- `CORE_AGENT_ALLOWED_ROOTS` (default `/tmp,/var/tmp,/home`)
- `CORE_AGENT_MAX_READ_BYTES` (default `1048576`)
- `CORE_AGENT_MAX_WRITE_BYTES` (default `1048576`)

## Implemented gRPC APIs

- `HealthService.Check`
- `SystemService.GetSystemOverview`
- `SystemService.GetRealtimeResource`
- `FileService.ListFiles`
- `FileService.ReadTextFile`
- `FileService.WriteTextFile`
- `FileService.CreateDirectory`
- `FileService.DeleteFile`

## Proto Source

- `../proto/agent/v1/agent.proto`
