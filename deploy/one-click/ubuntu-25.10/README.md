# One-Click Install (Ubuntu 25.10)

Language: **English** | [简体中文](README.zh-CN.md)

This installer is designed for **Ubuntu 25.10** and deploys SnowPanel in:

- host `core-agent` via systemd (recommended runtime mode)
- backend/frontend/postgres/redis via docker compose (`docker-compose.host-agent.yml`)

## Script

- `install.sh`

## Run

```bash
curl -fsSL https://raw.githubusercontent.com/snowfallx-bot/SnowPanel/main/deploy/one-click/ubuntu-25.10/install.sh -o install.sh
sudo bash install.sh
```

Or run from a local repository checkout:

```bash
sudo bash deploy/one-click/ubuntu-25.10/install.sh
```

## Optional Flags

```bash
sudo bash install.sh \
  --install-dir /opt/snowpanel \
  --branch main \
  --backend-port 8080 \
  --frontend-port 5173 \
  --admin-username admin \
  --admin-email admin@example.com \
  --docker-pull-retries 5
```

Important options:

- `--admin-password`: set bootstrap admin password explicitly
- `--jwt-secret`: set JWT secret explicitly
- `--docker-registry-mirror`: configure Docker daemon mirror (useful in regions with unstable Docker Hub access)
- `--docker-pull-retries`: retry count for image pull and compose up
- `--postgres-image` / `--redis-image`: override primary runtime images
- `--postgres-image-fallback` / `--redis-image-fallback`: override fallback images when primary pull fails
- `--force-unsupported`: allow non-Ubuntu-25.10 hosts (not recommended)

If `*-alpine` image pull keeps failing in your region, you can switch to non-alpine tags directly:

```bash
sudo bash install.sh --postgres-image postgres:16 --redis-image redis:7
```

The installer also includes an automatic fallback: if both primary and fallback pulls fail and Docker has `registry-mirrors` configured, it will temporarily remove mirrors and retry pulls once.

On every install run, the script also re-applies the baseline PostgreSQL schema in an idempotent way after `postgres` becomes healthy. This helps repeated installs and hosts that already have a persisted `postgres_data` volume.

## Output

Installer writes generated credentials to:

- `/root/.snowpanel/installer-output.env`

After install:

- Frontend: `http://127.0.0.1:<FRONTEND_PORT>`
- Backend health: `http://127.0.0.1:<BACKEND_PORT>/health`

The installer configures frontend API access in same-origin mode by default and lets the Vite container proxy `/api`, `/health`, and `/ready` to the backend service. This avoids remote-browser login failures caused by `127.0.0.1` pointing to the user's own machine.

## Notes

- For production tests, review and tighten:
  - `/etc/snowpanel/core-agent.env`
  - `${INSTALL_DIR}/.env`
- If backend health verification fails, the installer now prints `docker compose ps`, backend logs, and `core-agent` status/logs automatically before exiting.
- Restrict network access to backend/core-agent ports before internet exposure.
