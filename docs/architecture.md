# Architecture Draft

## Overview

SnowPanel uses a split architecture:
- `frontend`: React SPA for admin operations.
- `backend`: Go API service for business logic, auth, RBAC, audit, and task orchestration.
- `core-agent`: Rust system agent exposing controlled host operations over gRPC.

## Communication

- Frontend <-> Backend: REST + WebSocket
- Backend <-> Core Agent: gRPC

## Data Layer

- PostgreSQL for persistent business data
- Redis for cache and transient state

## Security Direction

- No arbitrary command execution interfaces
- Strict path and parameter validation for system/file operations
- Audit hooks for critical actions
