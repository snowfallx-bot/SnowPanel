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
- Keep metrics endpoint (`CORE_AGENT_METRICS_HOST:CORE_AGENT_METRICS_PORT`, default `127.0.0.1:9108`) on loopback or trusted scrape network only.
- Keep `CORE_AGENT_ALLOWED_ROOTS`, service whitelist, and cron allowlist minimal.
- If you can, bind `CORE_AGENT_HOST` to private interfaces instead of public `0.0.0.0`.
