# SnowPanel

[![CI](https://github.com/snowfallx-bot/SnowPanel/actions/workflows/ci.yml/badge.svg)](https://github.com/snowfallx-bot/SnowPanel/actions/workflows/ci.yml)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://www.mozilla.org/MPL/2.0/)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](backend/go.mod)
[![Gin](https://img.shields.io/badge/Gin-1.10-008ECF)](backend/go.mod)
[![GORM](https://img.shields.io/badge/GORM-1.30-0E5A8A)](backend/go.mod)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)](docker-compose.yml)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white)](docker-compose.yml)
[![JWT](https://img.shields.io/badge/JWT-v5-000000?logo=jsonwebtokens&logoColor=white)](backend/go.mod)
[![Rust](https://img.shields.io/badge/Rust-Edition%202021-000000?logo=rust&logoColor=white)](core-agent/Cargo.toml)
[![Tokio](https://img.shields.io/badge/Tokio-1.41-333333)](core-agent/Cargo.toml)
[![Tonic](https://img.shields.io/badge/Tonic-0.12-4A5568)](core-agent/Cargo.toml)
[![gRPC](https://img.shields.io/badge/gRPC-API-244C5A?logo=grpc&logoColor=white)](proto/README.md)
[![Protocol Buffers](https://img.shields.io/badge/Protocol_Buffers-Proto3-336791)](proto/agent/v1/agent.proto)
[![React](https://img.shields.io/badge/React-18.3-61DAFB?logo=react&logoColor=black)](frontend/package.json)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.8-3178C6?logo=typescript&logoColor=white)](frontend/package.json)
[![Vite](https://img.shields.io/badge/Vite-5.4-646CFF?logo=vite&logoColor=white)](frontend/package.json)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind_CSS-3.4-06B6D4?logo=tailwindcss&logoColor=white)](frontend/package.json)
[![TanStack Query](https://img.shields.io/badge/TanStack_Query-5.68-FF4154)](frontend/package.json)
[![Zustand](https://img.shields.io/badge/Zustand-5.0-5C4B51)](frontend/package.json)
[![Docker](https://img.shields.io/badge/Docker-Containerized-2496ED?logo=docker&logoColor=white)](docker-compose.yml)
[![Docker Compose](https://img.shields.io/badge/Docker_Compose-v2-1488C6?logo=docker&logoColor=white)](docker-compose.yml)

Language: **English** | [简体中文](README.zh-CN.md)

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

- [Architecture](docs/architecture.md) | [架构说明](docs/architecture.zh-CN.md)
- [Development Guide](docs/development.md) | [开发指南](docs/development.zh-CN.md)
- [API Design](docs/api-design.md) | [API 设计](docs/api-design.zh-CN.md)
- [Security Notes](docs/security.md) | [安全说明](docs/security.zh-CN.md)
- [Deployment Guide](docs/deployment.md) | [部署指南](docs/deployment.zh-CN.md)
- [Roadmap](docs/roadmap.md) | [路线图草案](docs/roadmap.zh-CN.md)

## Contributing And Community

- [Contributing Guide](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md)
- [Issue Templates](.github/ISSUE_TEMPLATE)
- [Pull Request Template](.github/pull_request_template.md)

## License

This project is licensed under the [Mozilla Public License 2.0](LICENSE).
