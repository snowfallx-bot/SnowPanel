.PHONY: up down logs up-host-agent down-host-agent logs-host-agent up-observability down-observability logs-observability up-host-agent-observability down-host-agent-observability logs-host-agent-observability backend agent frontend lint test proto-go

PROTO_SRC := proto/agent/v1/agent.proto
PROTO_GO_OUT := backend/internal/grpcclient/pb

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

up-observability:
	docker compose -f docker-compose.yml -f docker-compose.observability.yml up -d --build

down-observability:
	docker compose -f docker-compose.yml -f docker-compose.observability.yml down

logs-observability:
	docker compose -f docker-compose.yml -f docker-compose.observability.yml logs -f --tail=200 prometheus alertmanager otel-collector jaeger

up-host-agent-observability:
	docker compose -f docker-compose.yml -f docker-compose.host-agent.yml -f docker-compose.observability.yml up -d --build

down-host-agent-observability:
	docker compose -f docker-compose.yml -f docker-compose.host-agent.yml -f docker-compose.observability.yml down

logs-host-agent-observability:
	docker compose -f docker-compose.yml -f docker-compose.host-agent.yml -f docker-compose.observability.yml logs -f --tail=200 prometheus alertmanager otel-collector jaeger

backend:
	cd backend && go run ./cmd/server

agent:
	cd core-agent && cargo run

frontend:
	cd frontend && npm run dev

proto-go:
	protoc \
	  --proto_path=. \
	  --go_out=paths=source_relative:$(PROTO_GO_OUT) \
	  --go-grpc_out=paths=source_relative:$(PROTO_GO_OUT) \
	  $(PROTO_SRC)

lint:
	cd backend && go vet ./...
	cd core-agent && cargo fmt --all -- --check
	cd frontend && npm run build

test:
	cd backend && go test ./...
	cd core-agent && cargo test
	cd frontend && npm run test
	cd frontend && npm run build
