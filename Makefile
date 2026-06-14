.PHONY: up down build migrate proto test-e2e

# Build services sequentially to avoid OOM-killing the Go compiler.
# Each service pulls gRPC + pgx which is heavy; parallel builds exhaust RAM.
up: build
	docker compose up

build:
	docker compose build --no-cache api-gateway
	docker compose build --no-cache sandbox-engine
	docker compose build --no-cache bot-fleet
	docker compose build --no-cache telemetry-ingester
	docker compose build --no-cache frontend

# Quick rebuild (uses layer cache) then start — use this after the first build
run:
	docker compose up --build

down:
	docker ps -aq --filter "name=submission-" | xargs -r docker rm -f
	docker compose down -v

migrate:
	bash migrations/run_migrations.sh

proto:
	bash scripts/gen_proto.sh

test-e2e:
	bash scripts/test_e2e.sh

# Run locally without Docker (for fast iteration)
dev-gateway:
	cd services/api-gateway && go run ./...

dev-sandbox:
	cd services/sandbox-engine && go run ./...

dev-bots:
	cd services/bot-fleet && go run ./...

dev-telemetry:
	cd services/telemetry-ingester && go run ./...
