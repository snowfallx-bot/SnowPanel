# Development Draft

## Prerequisites

- Go 1.25+
- Rust stable toolchain
- Node.js 22+
- Docker Desktop

## Local Start (Draft)

1. `cp .env.example .env`
2. `docker compose up -d postgres redis`
3. Backend: `cd backend && go run ./cmd/server`
4. Agent: `cd core-agent && cargo run`
5. Frontend: `cd frontend && npm install && npm run dev`

## Conventions

- Keep controllers/handlers thin
- Use service/repository layering
- Preserve secure-by-default behavior for host operations
