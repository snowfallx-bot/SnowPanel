.PHONY: up down logs up-host-agent down-host-agent logs-host-agent up-observability down-observability logs-observability up-host-agent-observability down-host-agent-observability logs-host-agent-observability backend agent frontend lint test proto-go

PROTO_SRC := proto/agent/v1/agent.proto
PROTO_GO_OUT := backend/internal/grpcclient/pb

CARGO_DEFAULT := cargo
NPM_DEFAULT := npm
PROTOC_DEFAULT := protoc
GO_PLUGIN_BIN_DEFAULT :=
ifeq ($(OS),Windows_NT)
	CARGO_HOME_BIN := $(subst \,/,$(USERPROFILE))/.cargo/bin/cargo.exe
	PROTOC_CANDIDATES := $(wildcard $(subst \,/,$(USERPROFILE))/.cargo/registry/src/*/protoc-bin-vendored-win32-*/bin/protoc.exe)
	ifneq ($(wildcard $(CARGO_HOME_BIN)),)
		CARGO_DEFAULT := $(CARGO_HOME_BIN)
	endif
	ifneq ($(PROTOC_CANDIDATES),)
		PROTOC_DEFAULT := $(firstword $(PROTOC_CANDIDATES))
	endif
	NPM_DEFAULT := npm.cmd
	GO_PLUGIN_BIN_DEFAULT := $(subst \,/,$(USERPROFILE))/go/bin
endif
CARGO ?= $(CARGO_DEFAULT)
NPM ?= $(NPM_DEFAULT)
PROTOC ?= $(PROTOC_DEFAULT)
GO_PLUGIN_BIN ?= $(GO_PLUGIN_BIN_DEFAULT)

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
	cd frontend && $(NPM) run dev

proto-go:
	PATH="$(GO_PLUGIN_BIN):$$PATH" $(PROTOC) \
	  --proto_path=. \
	  --go_out=paths=source_relative:$(PROTO_GO_OUT) \
	  --go-grpc_out=paths=source_relative:$(PROTO_GO_OUT) \
	  $(PROTO_SRC)

lint:
	cd backend && go vet ./...
	cd core-agent && $(CARGO) fmt --all -- --check
	cd frontend && $(NPM) run build

test:
	cd backend && go test ./...
	cd core-agent && $(CARGO) test
	cd frontend && $(NPM) run test
	cd frontend && $(NPM) run build
