FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY scripts/test_submission/ ./scripts/test_submission/
WORKDIR /app/scripts/test_submission
RUN go build -o /bin/test-sub ./main.go

FROM alpine:3.19
RUN adduser -D -u 1001 appuser
COPY --from=builder /bin/test-sub /bin/test-sub
USER appuser
EXPOSE 8080
ENTRYPOINT ["/bin/test-sub"]
