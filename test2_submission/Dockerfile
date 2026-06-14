FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /bin/server ./main.go

FROM alpine:3.19
RUN adduser -D -u 1001 appuser
COPY --from=builder /bin/server /bin/server
USER appuser
EXPOSE 8080
ENTRYPOINT ["/bin/server"]
