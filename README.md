# Bench Platform - Distributed Benchmarking

This repository contains the complete **Day 1 Baseline** for the IICPC Summer Hackathon 2026 distributed benchmarking platform.

## 🚀 Day 1 Accomplishments

We have successfully built and integrated the entire end-to-end microservices architecture:
- **Frontend (`React / Vite`)**: A premium, light-themed SaaS interface with a custom SVG logo, live SSE leaderboard, and seamless file uploading.
- **API Gateway (`Go`)**: The central entrypoint handling zip uploads, authentication, CORS, and orchestrating status flows.
- **Sandbox Engine (`Go`)**: Dynamically builds user-submitted trading engines in isolated Docker containers, executes health checks, and manages lifecycle watchdogs.
- **Bot Fleet (`Go`)**: Generates rapid, asynchronous load and adversarial trading scenarios against the submitted sandbox engines.
- **Telemetry Ingester (`Go`)**: Ingests massive volumes of gRPC metrics from the Bot Fleet in real-time, calculates correctness, aggregates P99 latency, and computes the final benchmark score.
- **Infrastructure**: A complete `docker-compose` orchestration featuring TimescaleDB (Postgres) and Redis.

## 🛠️ How to Run

1. **Start the Stack**: 
   ```bash
   docker compose up -d --build
   ```
2. **Access the Frontend**: Navigate to `http://localhost:3000`
3. **Run a Benchmark**: 
   - Enter a Team Name and Token.
   - Upload your `test_sub.zip` containing your Dockerfile and source code.
   - Watch the SSE stream update your UPLOADED -> BUILDING -> RUNNING -> BENCHMARKING -> SCORED phases in real-time!

## 📂 Repository Layout

- `services/frontend/` - React SPA (Vite, Nginx)
- `services/api-gateway/` - Core API (Go)
- `services/sandbox-engine/` - Docker orchestration (Go)
- `services/bot-fleet/` - Load generator (Go)
- `services/telemetry-ingester/` - Metrics & Scoring engine (Go)
- `shared/` - Frozen data contracts and generated gRPC protobufs
- `scripts/` - Utilities for e2e testing and proto generation
