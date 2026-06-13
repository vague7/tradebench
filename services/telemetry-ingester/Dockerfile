FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum ./
COPY shared/ ./shared/
COPY services/telemetry-ingester/ ./services/telemetry-ingester/

RUN apk add --no-cache g++ make
RUN make -C /app/services/telemetry-ingester/correctness/reference
WORKDIR /app/services/telemetry-ingester
RUN go build -o /bin/service ./...

# Stage 2: minimal runtime
FROM alpine:3.19
RUN apk add --no-cache libstdc++
RUN adduser -D -u 1001 appuser
WORKDIR /app
COPY --from=builder /bin/service /bin/service
COPY --from=builder /app/services/telemetry-ingester/correctness/reference/bin/ref-engine /app/correctness/reference/bin/ref-engine
USER appuser
EXPOSE 9003
ENTRYPOINT ["/bin/service"]
