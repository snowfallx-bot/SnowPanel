.PHONY: up down backend agent frontend

up:
	docker compose up -d postgres redis

down:
	docker compose down

backend:
	cd backend && go run ./cmd/server

agent:
	cd core-agent && cargo run

frontend:
	cd frontend && npm run dev
