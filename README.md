# Tradebench Platform

Tradebench is a distributed load-testing and benchmarking platform for evaluating trading exchange implementations. It allows teams to upload their trading engine as a Docker container, securely builds and deploys it in an isolated sandbox, and subjects it to massive concurrent load testing using an automated bot fleet.

## Quick Start

### 1. Prerequisites
- Docker and Docker Compose
- Node.js (for local frontend development)

### 2. Running the Platform

To start the entire platform (Postgres, Redis, API Gateway, Sandbox Engine, Bot Fleet, Telemetry Ingester, and the React Frontend):

```bash
docker-compose up --build -d
```

The frontend will be available at [http://localhost:3000](http://localhost:3000).

### 3. Submitting an Exchange

To submit an exchange, upload a ZIP file containing a `Dockerfile` and your source code via the web UI.

The platform will:
1. Extract and build your Docker container
2. Deploy it securely on the isolated `bench-net`
3. Hit it with massive load from the `bot-fleet`
4. Calculate your Final Score based on Throughput, Latency, and Correctness
5. Publish your rank to the real-time Leaderboard

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture diagrams, component trees, and network isolation details.

## Development

The project is split into several microservices:
- `services/api-gateway`: Go REST API for uploads and leaderboard (port 8080)
- `services/sandbox-engine`: Go service orchestrating isolated Docker containers
- `services/bot-fleet`: Go distributed load generator
- `services/telemetry-ingester`: Go metrics aggregator
- `services/frontend`: React/Vite SPA (port 3000)

## Authors
Built by the Tradebench Engineering Team.
