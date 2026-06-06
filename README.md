# Bench Platform

Repository baseline for the IICPC Summer Hackathon 2026 distributed benchmarking platform.

## Repository Status

The initial scaffold is complete and organized for the three-engineer handoff. The repo is now ready to push to GitHub, and each engineer can branch from `main` and work on their own target area.



## Layout

- `shared/` frozen contracts shared across services
- `shared/proto/` gRPC interface definitions
- `migrations/` append-only database migrations
- `scripts/gen_proto.sh` proto regeneration entrypoint

## Notes

- Go version: 1.22
- Shared contracts are frozen after Day Zero unless all engineers agree on a schema change.
- Service scaffolds are added in later phases.

## Day 1 Ownership

### Engineer 1

- Build the `api-gateway` upload, status, results, leaderboard, and admin endpoints.
- Implement auth middleware, upload size enforcement, and the database/Redis store layer.
- Finish the sandbox runner flow in `sandbox-engine`, including Docker build, container spawn, health checks, and watchdog cleanup.
- Commit the initial migration files and keep schema changes append-only.

### Engineer 2

- Implement `bot-fleet` load generation, adversarial scenarios, and gRPC streaming to telemetry.
- Implement `telemetry-ingester` buffering, aggregation, scoring, correctness validation, and DB writes.
- Finalize the composite score formula and the reference-engine integration path.

### Engineer 3

- Build the React/Vite frontend for submission upload and status polling.
- Add the live leaderboard SSE view and the typed API client wiring.
- Maintain the Docker Compose integration, frontend build, and the dummy submission used for end-to-end checks.

### Working Agreement

- All engineers branch from `main` after this scaffold is pushed.
- Shared contracts in `shared/` remain frozen unless all three engineers agree to a schema change.
- New work should stay inside the ownership boundaries defined in the PRD.
