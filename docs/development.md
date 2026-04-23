# Development Guide

Language: **English** | [简体中文](development.zh-CN.md)

## Prerequisites

- Docker + Docker Compose v2
- Go 1.25+
- Rust stable toolchain
- Node.js 22+

## One-Command Local Stack

1. Copy environment template:
   - `cp .env.example .env`
2. Start full stack:
   - `make up`
3. Access services:
   - Frontend: `http://127.0.0.1:5173`
   - Backend health: `http://127.0.0.1:8080/health`
   - Backend readiness: `http://127.0.0.1:8080/ready`
4. Stop stack:
   - `make down`

## Split Local Workflow

When you want local binaries + containerized dependencies:

1. Start PostgreSQL and Redis:
   - `docker compose up -d postgres redis`
   - for local backend binaries, publish dependency ports with:
     `docker compose -f docker-compose.yml -f docker-compose.local.yml up -d postgres redis`
2. Run core-agent:
   - `make agent`
3. Run backend:
   - `make backend`
4. Run frontend:
   - `make frontend`

## Useful Commands

- `make logs`: tail compose logs
- `make lint`: baseline static checks (`go vet`, `cargo fmt --check`, frontend build)
- `make test`: backend tests + rust tests + frontend test/build flow

## Test Coverage Scope

Current minimum test suite focuses on:
- backend auth service and auth/permission middleware behavior.
- core-agent path validator and system info service sanity.
- frontend login page render baseline.

## Coding Conventions

- Handlers should stay focused on binding/response only.
- Business logic belongs to `service` layer.
- DB operations belong to `repository` layer.
- Host-facing operations must stay explicit and validated; no arbitrary command execution.
