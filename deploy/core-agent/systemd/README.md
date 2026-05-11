# Core-Agent Systemd Deployment

Language: **English** | [简体中文](README.zh-CN.md)

This folder provides a baseline host deployment template for running `core-agent` as a native systemd service.

## Files

- `core-agent.service`: systemd unit template.
- `core-agent.env.example`: environment variable template loaded by systemd.

## Install Steps (Linux host)

1. Build binary on target host:
   - `cd core-agent`
   - `cargo build --release`
2. Install binary and config:
   - `sudo install -Dm755 target/release/core-agent /usr/local/bin/core-agent`
   - `sudo install -d -m 750 /etc/snowpanel`
   - `sudo install -Dm640 deploy/core-agent/systemd/core-agent.env.example /etc/snowpanel/core-agent.env`
3. Install and start systemd unit:
   - `sudo install -Dm644 deploy/core-agent/systemd/core-agent.service /etc/systemd/system/core-agent.service`
   - `sudo systemctl daemon-reload`
   - `sudo systemctl enable --now core-agent`
4. Verify:
   - `sudo systemctl status core-agent --no-pager`
   - `ss -lntp | grep 50051`
   - `curl -fsS http://127.0.0.1:9108/metrics | head`

If you also want distributed tracing from host `core-agent`, enable these in `/etc/snowpanel/core-agent.env`:

- `OTEL_TRACING_ENABLED=true`
- `OTEL_SERVICE_NAME=snowpanel-core-agent`
- `OTEL_EXPORTER_OTLP_ENDPOINT=<collector-host>:4317`

For production backend authentication to host `core-agent`, keep `CORE_AGENT_AUTH_MODE=token`, set a strong `CORE_AGENT_SHARED_TOKEN`, and configure backend with the same value in `BACKEND_AGENT_SHARED_TOKEN`.

Production `core-agent` startup is deny-by-default unless `CORE_AGENT_ALLOW_UNSAFE_PRODUCTION_CONFIG=true` is explicitly set. In production, keep:

- `CORE_AGENT_AUTH_MODE=token`
- `CORE_AGENT_ALLOWED_ROOTS` limited to concrete directories, never `/`
- `CORE_AGENT_SERVICE_WHITELIST` non-empty when service operations are enabled
- `CORE_AGENT_CRON_ALLOWED_COMMANDS` explicitly configured when cron operations are enabled

Disable unused operation categories with:

- `CORE_AGENT_ENABLE_FILE_OPS=false`
- `CORE_AGENT_ENABLE_SERVICE_OPS=false`
- `CORE_AGENT_ENABLE_DOCKER_OPS=false`
- `CORE_AGENT_ENABLE_CRON_OPS=false`

## Backend Compose with Host Agent

When backend runs in Docker but `core-agent` runs on host, use:

- `make up-host-agent`

This override points backend to `host.docker.internal:50051` and disables the containerized `core-agent` service by default.

For later rebuilds or log inspection, keep using:

- `make up-host-agent`
- `make logs-host-agent`

Do not fall back to plain `docker compose up` / `make up`, or backend will reconnect to the containerized `core-agent` instead of the host systemd service.

## Security Notes

- Restrict network access to port `50051` (firewall / private network only).
- Keep `CORE_AGENT_AUTH_MODE=token` in production until mTLS support is implemented, and never log or publish `CORE_AGENT_SHARED_TOKEN`.
- Keep metrics endpoint (`CORE_AGENT_METRICS_HOST:CORE_AGENT_METRICS_PORT`, default `127.0.0.1:9108`) on loopback or trusted scrape network only.
- Keep OTLP export endpoint limited to trusted collector/backends only.
- Keep `CORE_AGENT_ALLOWED_ROOTS`, service whitelist, and cron allowlist minimal.
- If you can, bind `CORE_AGENT_HOST` to private interfaces instead of public `0.0.0.0`.
- Back up host configuration and application data before enabling destructive file, service, Docker, or cron operations.

## Systemd Hardening

The template enables `NoNewPrivileges=true`, `PrivateTmp=true`, `ProtectSystem=full`, `ProtectHome=read-only`, kernel/control-group protections, explicit `ReadWritePaths`, and `RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX`.

Tradeoffs:

- `ProtectSystem=strict` is stronger, but can conflict with Docker socket and systemd interaction on some hosts. Start with `full`, then test stricter settings per distribution.
- `ReadWritePaths` must match `CORE_AGENT_ALLOWED_ROOTS` plus runtime socket paths such as `/run` or `/var/run` when Docker operations are enabled.
- `CapabilityBoundingSet` is not constrained in the baseline template because file ownership and host service workflows vary by distribution. Add a host-specific minimal bounding set only after validating file, Docker, and systemd operations.
