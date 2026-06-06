FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum ./
COPY shared/ ./shared/
COPY services/bot-fleet/ ./services/bot-fleet/
WORKDIR /app/services/bot-fleet
RUN go build -o /bin/service ./...

FROM alpine:3.19
RUN adduser -D -u 1001 appuser
COPY --from=builder /bin/service /bin/service
USER appuser
ENTRYPOINT ["/bin/service"]
