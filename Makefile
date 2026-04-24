.PHONY: up down logs up-host-agent down-host-agent logs-host-agent backend agent frontend lint test

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f --tail=200

up-host-agent:
	docker compose -f docker-compose.yml -f docker-compose.host-agent.yml up -d --build

down-host-agent:
	docker compose -f docker-compose.yml -f docker-compose.host-agent.yml down

logs-host-agent:
	docker compose -f docker-compose.yml -f docker-compose.host-agent.yml logs -f --tail=200

backend:
	cd backend && go run ./cmd/server

agent:
	cd core-agent && cargo run

frontend:
	cd frontend && npm run dev

lint:
	cd backend && go vet ./...
	cd core-agent && cargo fmt --all -- --check
	cd frontend && npm run build

test:
	cd backend && go test ./...
	cd core-agent && cargo test
	cd frontend && npm run test
	cd frontend && npm run build
