# SnowPanel

SnowPanel is a Linux panel prototype (similar to BT Panel / 1Panel) built as a monorepo.

## Repository Layout

```text
.
├── backend      # Go HTTP API service
├── core-agent   # Rust system agent
├── deploy       # Deployment scripts and manifests
├── docs         # Project documentation
└── frontend     # React + TypeScript + Vite web UI
```

## Quick Start

1. Copy environment variables:
   `cp .env.example .env`
2. Start local dependencies:
   `docker compose up -d postgres redis`
3. Start backend:
   `cd backend && go run ./cmd/server`
4. Start core-agent:
   `cd core-agent && cargo run`
5. Start frontend:
   `cd frontend && npm install && npm run dev`

## Current Stage

The repository currently contains stage-1 bootstrap scaffolding:
- monorepo directories
- runnable minimal backend server
- runnable minimal core-agent skeleton
- React + TypeScript + Vite frontend with base routing/layout
- initial docs drafts
