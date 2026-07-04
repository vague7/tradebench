# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum ./
COPY shared/ ./shared/
COPY services/api-gateway/go.mod        ./services/api-gateway/go.mod
COPY services/sandbox-engine/go.mod     ./services/sandbox-engine/go.mod
COPY services/telemetry-ingester/go.mod ./services/telemetry-ingester/go.mod
COPY services/bot-fleet/ ./services/bot-fleet/
WORKDIR /app/services/bot-fleet
RUN --mount=type=cache,target=/root/.cache/go-build,id=gobuild,sharing=locked \
    --mount=type=cache,target=/go/pkg/mod,id=gomod,sharing=locked \
    go build -ldflags="-s -w" -trimpath -o /bin/service .

FROM alpine:3.19
RUN adduser -D -u 1001 appuser
COPY --from=builder /bin/service /bin/service
USER appuser
EXPOSE 9002
ENTRYPOINT ["/bin/service"]
