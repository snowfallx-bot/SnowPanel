# SnowPanel

[![CI](https://github.com/snowfallx-bot/SnowPanel/actions/workflows/ci.yml/badge.svg)](https://github.com/snowfallx-bot/SnowPanel/actions/workflows/ci.yml)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://www.mozilla.org/MPL/2.0/)

SnowPanel is a Linux server panel prototype (in the spirit of BT Panel / 1Panel) built as a monorepo.  
It is split into clear service boundaries so the UI/API layer and host-control layer can evolve independently.

## Architecture

| Component | Stack | Responsibility |
| --- | --- | --- |
| `frontend` | React + TypeScript + Vite | Admin UI |
| `backend` | Go + Gin + GORM + JWT + RBAC | HTTP API and business logic |
| `core-agent` | Rust + gRPC | Host capability layer (files, services, docker, cron) |
| `proto` | Protocol Buffers | Shared gRPC contracts |
| `docs` | Markdown | Architecture, development, security, deployment notes |

## Repository Layout

```text
.
├── backend      # Go API service
├── core-agent   # Rust gRPC agent
├── deploy       # Deployment notes
├── docs         # Architecture and development docs
├── frontend     # React admin UI
└── proto        # Shared gRPC proto definitions
```

## Requirements

- Docker + Docker Compose v2 (recommended workflow)
- Optional local toolchains:
  - Go `1.25+`
  - Rust stable
  - Node.js `22+`

## Quick Start

1. Copy environment file:
   - macOS/Linux: `cp .env.example .env`
   - PowerShell: `Copy-Item .env.example .env`
2. Start all services:
   - `make up`
3. Open:
   - Frontend: `http://127.0.0.1:5173`
   - Backend health: `http://127.0.0.1:8080/health`
4. Default admin (auto-bootstrapped when DB is empty):
   - username: `admin`
   - password: `admin123456`
5. Stop services:
   - `make down`

## Local Development

1. Start dependencies only:
   - `docker compose up -d postgres redis`
2. Run each service locally:
   - `make agent`
   - `make backend`
   - `make frontend`

Common commands:

- `make up`: start all services with build
- `make down`: stop all services
- `make logs`: tail compose logs
- `make lint`: baseline static checks
- `make test`: backend/core-agent tests + frontend test/build checks

## Documentation

- [Architecture](docs/architecture.md)
- [Development Guide](docs/development.md)
- [Security Notes](docs/security.md)
- [Deployment Notes](docs/deployment.md)
- [Roadmap](docs/roadmap.md)

## Contributing And Community

- [Contributing Guide](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md)
- [Issue Templates](.github/ISSUE_TEMPLATE)
- [Pull Request Template](.github/pull_request_template.md)

## License

This project is licensed under the [Mozilla Public License 2.0](LICENSE).
