FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum ./
COPY shared/ ./shared/
COPY services/ ./services/
WORKDIR /app/services/sandbox-engine
RUN go build -ldflags="-s -w" -trimpath -o /bin/service .

FROM alpine:3.19
RUN adduser -D -u 1001 appuser
# Create the submissions directory owned by appuser before switching user
RUN mkdir -p /data/submissions && chown -R appuser:appuser /data
COPY --from=builder /bin/service /bin/service
USER appuser
EXPOSE 9001
ENTRYPOINT ["/bin/service"]
