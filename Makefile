.PHONY: up down build migrate proto test-e2e

up:
	docker compose up --build

down:
	docker compose down -v

build:
	docker compose build

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
