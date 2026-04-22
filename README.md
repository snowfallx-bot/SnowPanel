# SnowPanel

SnowPanel is a Linux panel prototype (similar to BT Panel / 1Panel) built as a monorepo:
- `core-agent`: Rust system/host capability layer (gRPC)
- `backend`: Go API layer (Gin + GORM + JWT + RBAC)
- `frontend`: React + TypeScript + Vite admin panel

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

## Prerequisites

- Docker + Docker Compose v2
- Optional local toolchain for non-container workflow:
  - Go 1.25+
  - Rust 1.82+
  - Node.js 22+

## Quick Start (Docker Compose)

1. Copy env:
   - `cp .env.example .env`
2. Start all services:
   - `make up`
3. Open:
   - Frontend: `http://127.0.0.1:5173`
   - Backend health: `http://127.0.0.1:8080/health`
4. Stop:
   - `make down`

Notes:
- PostgreSQL schema is initialized on first startup via `backend/migrations/0001_init_schema.up.sql` mounted to `docker-entrypoint-initdb.d`.
- Default admin is auto-bootstrapped when database has no users:
  - username: `admin`
  - password: `admin123456`

## Local Development (Without Containers)

1. Start dependencies only:
   - `docker compose up -d postgres redis`
2. Start core-agent:
   - `make agent`
3. Start backend:
   - `make backend`
4. Start frontend:
   - `make frontend`

## Common Make Commands

- `make up`: start all services with build
- `make down`: stop all services
- `make logs`: tail compose logs
- `make backend`: run backend locally
- `make agent`: run core-agent locally
- `make frontend`: run frontend locally
- `make lint`: run baseline static checks
- `make test`: run backend/core-agent tests and frontend build check

## Common Issues

- `relation "users" does not exist`:
  - Usually old postgres volume or first boot did not finish initialization. Run `make down`, remove volume, then `make up`.
- Backend cannot connect to agent:
  - Verify `AGENT_TARGET` and ensure `core-agent` is running (`docker compose ps`).
- Frontend login/API request fails:
  - Check `VITE_API_BASE_URL` in `.env` (default `http://127.0.0.1:8080`).
