# P3-0 Stabilization Report

Date: 2026-05-09

Base commit under test: `2be39ca`

Branch: `p3-production-hardening`

## Environment

- OS: Microsoft Windows NT 10.0.26100.0
- PowerShell: 7.5.4
- GNU Make: 4.4.1
- Go: go1.25.3 windows/amd64
- Cargo: 1.95.0
- rustc: 1.95.0
- Node.js: v22.18.0
- npm: 10.9.3
- Docker CLI: 29.1.3
- Docker Compose: v2.40.3-desktop.1
- protoc: libprotoc 31.1, from cached `protoc-bin-vendored-win32`
- protoc-gen-go: v1.36.5
- protoc-gen-go-grpc: v1.5.1

## Commands Run

| Command | Result | Notes |
| --- | --- | --- |
| `make lint` | Pass | Required elevated run in this local sandbox because frontend build under MSYS attempted to read outside the workspace. |
| `make test` | Pass | Backend, core-agent, frontend tests, and frontend build passed. |
| `cd backend && go test ./...` | Pass | All backend packages passed or had no test files. |
| `cd core-agent && cargo fmt --all -- --check` | Pass | Ran with explicit cargo path because `cargo` is not on PATH in this Windows shell. |
| `cd core-agent && cargo test` | Pass | 25 tests passed. |
| `cd frontend && npm ci` | Pass | Installed 264 packages from lockfile. |
| `cd frontend && npm run test` | Pass | 6 files and 24 tests passed. |
| `cd frontend && npm run build` | Pass | Vite production build completed. |
| `make proto-go` | Pass | Uses cached vendored `protoc.exe` and Go plugins from `C:\Users\GuaiZai\go\bin`. |
| `git diff --exit-code -- backend/internal/grpcclient/pb/proto/agent/v1/agent.pb.go backend/internal/grpcclient/pb/proto/agent/v1/agent_grpc.pb.go` | Fail before fix | Generated files drifted after running current local proto tooling. The generated files were updated in this milestone. |
| `make up` | Pass | Compose mode built and started backend, frontend, postgres, redis, and container core-agent. |
| `curl.exe -f http://127.0.0.1:8080/health` | Pass | Compose mode returned `agent=up` and `database=up`. |
| `curl.exe -f http://127.0.0.1:8080/ready` | Pass | Compose mode returned `status=ready`, `agent=up`, and `database=up`. |
| `make down` | Pass | Compose mode was stopped and cleaned up. |
| `docker info` | Pass after Docker Desktop start | Initial run failed because the Docker Desktop Linux engine was not running. |
| `make up-host-agent` | Pass | Host-agent compose mode started backend, frontend, postgres, and redis. |
| host `core-agent.exe` on `0.0.0.0:50051` | Pass | Started from `core-agent/target/debug/core-agent.exe` for local host-agent smoke. |
| `curl.exe -f http://127.0.0.1:8080/health` | Pass | Host-agent mode returned `agent=up` and `database=up`. |
| `curl.exe -f http://127.0.0.1:8080/ready` | Pass | Host-agent mode returned `status=ready`, `agent=up`, and `database=up`. |
| `make down-host-agent` | Pass | Host-agent compose mode was stopped and cleaned up. |

## Fixed Failures

- Updated `Makefile` so Windows local runs can find `%USERPROFILE%/.cargo/bin/cargo.exe`.
- Updated `Makefile` so Windows local runs use `npm.cmd` instead of the extensionless `npm` shim under MSYS.
- Updated `Makefile` so `make proto-go` can use cached vendored `protoc.exe` and the local Go protobuf plugins.
- Regenerated checked-in Go protobuf bindings with the available local proto toolchain.
- Started Docker Desktop after the initial daemon check failed.
- Retried Docker image pulls after an initial transient Docker Hub EOF while resolving `redis:7-alpine`.
- Started a local host core-agent process for host-agent smoke because host-agent compose mode expects `host.docker.internal:50051`.

## Compose Smoke Evidence

Compose smoke passed.

```powershell
make up
curl.exe -f http://127.0.0.1:8080/health
curl.exe -f http://127.0.0.1:8080/ready
make down
```

Health response:

```json
{"code":0,"message":"ok","data":{"checks":{"agent":"up","database":"up"},"service":"backend","status":"up"}}
```

Readiness response:

```json
{"code":0,"message":"ok","data":{"checks":{"agent":"up","database":"up"},"service":"backend","status":"ready"}}
```

## Host-Agent Smoke Evidence

Host-agent smoke passed after starting a local host core-agent process on `0.0.0.0:50051`.

```powershell
C:\Users\GuaiZai\.cargo\bin\cargo.exe build
# Started core-agent/target/debug/core-agent.exe with:
# CORE_AGENT_HOST=0.0.0.0
# CORE_AGENT_PORT=50051
# CORE_AGENT_METRICS_ENABLED=false
make up-host-agent
curl.exe -f http://127.0.0.1:8080/health
curl.exe -f http://127.0.0.1:8080/ready
make down-host-agent
```

Health response:

```json
{"code":0,"message":"ok","data":{"checks":{"agent":"up","database":"up"},"service":"backend","status":"up"}}
```

Readiness response:

```json
{"code":0,"message":"ok","data":{"checks":{"agent":"up","database":"up"},"service":"backend","status":"ready"}}
```

## Unresolved Risks

- Docker Desktop must be running before local compose gates; otherwise Docker API checks fail against `npipe:////./pipe/dockerDesktopLinuxEngine`.
- Docker CLI can report `C:\Users\GuaiZai\.docker\config.json` access denied in non-elevated checks; elevated Docker checks were used for local validation on this machine.
- `cargo` and `rustc` are installed but not on PATH; this report used explicit paths or Makefile auto-discovery.
- Frontend tests emit Vite deprecation warnings for esbuild options inherited from the current toolchain; they do not fail the gate.
