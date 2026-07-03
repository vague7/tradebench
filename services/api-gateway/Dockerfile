# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum ./
COPY shared/ ./shared/
COPY services/bot-fleet/go.mod          ./services/bot-fleet/go.mod
COPY services/sandbox-engine/go.mod     ./services/sandbox-engine/go.mod
COPY services/telemetry-ingester/go.mod ./services/telemetry-ingester/go.mod
COPY services/api-gateway/ ./services/api-gateway/
WORKDIR /app/services/api-gateway
RUN --mount=type=cache,target=/root/.cache/go-build,id=gobuild,sharing=locked \
    --mount=type=cache,target=/go/pkg/mod,id=gomod,sharing=locked \
    go build -ldflags="-s -w" -trimpath -o /bin/service .

FROM alpine:3.19
RUN adduser -D -u 1001 appuser
RUN mkdir -p /data/submissions && chown -R appuser:appuser /data
COPY --from=builder /bin/service /bin/service
USER appuser
EXPOSE 8080
ENTRYPOINT ["/bin/service"]
