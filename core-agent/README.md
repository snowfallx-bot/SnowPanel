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

## Implemented gRPC APIs

- `HealthService.Check`
- `SystemService.GetSystemOverview`
- `SystemService.GetRealtimeResource`

## Proto Source

- `../proto/agent/v1/agent.proto`
