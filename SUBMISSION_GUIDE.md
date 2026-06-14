# How to Submit Your Exchange Engine

## What to Build

Your submission must be a ZIP file containing a Go (or any language) HTTP server that implements a simple exchange API on port 8080.

## Required Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Returns `{"status":"ok"}` with HTTP 200 |
| `POST` | `/order` | Accept a new order, return order ID |
| `DELETE` | `/order/:id` | Cancel an order by ID |
| `GET` | `/orderbook` | Return current bids and asks |

### POST /order — Request Body
```json
{
  "type": "LIMIT",
  "side": "BUY",
  "price": 100.50,
  "qty": 10.0
}
```
`type` is one of: `LIMIT`, `MARKET`, `CANCEL`

### POST /order — Response
```json
{
  "orderId": "your-unique-order-id",
  "status": "ACCEPTED"
}
```

### DELETE /order/:id — Response
```json
{
  "orderId": "the-cancelled-id",
  "status": "CANCELLED"
}
```

### GET /orderbook — Response
```json
{
  "bids": [],
  "asks": []
}
```

---

## Required Files in ZIP

Your ZIP must contain these three files **at the root level** (not inside a subfolder):

```
submission.zip
├── Dockerfile       ← required, must be named exactly "Dockerfile"
├── main.go          ← your server code (or any language equivalent)
└── go.mod           ← Go module file (if using Go)
```

### Dockerfile Requirements

- Must expose port **8080**
- Must start your server as the entrypoint
- Must use a base image compatible with Linux/amd64

**Example Dockerfile (Go):**
```dockerfile
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
```

**Example go.mod:**
```
module my-exchange

go 1.22
```

---

## How to Create the ZIP

```bash
# Navigate INTO your submission folder first
cd my-submission/

# Verify these files exist at the root
ls
# Dockerfile  main.go  go.mod

# Create the ZIP
zip submission.zip Dockerfile main.go go.mod

# Verify the zip contents (Dockerfile must be at root, not in a subfolder)
unzip -l submission.zip
# Archive:  submission.zip
#   Length   Name
#   -------  ----
#            Dockerfile    ← must appear like this, NOT as subfolder/Dockerfile
#            main.go
#            go.mod
```

### ❌ Common Mistakes

```bash
# WRONG — zipping from outside the folder creates nested paths
zip submission.zip my-submission/Dockerfile my-submission/main.go
# Results in: my-submission/Dockerfile (Docker can't find it)

# WRONG — using -r on the folder itself
zip -r submission.zip my-submission/
# Results in: my-submission/Dockerfile (Docker can't find it)

# CORRECT — cd into the folder first
cd my-submission && zip submission.zip Dockerfile main.go go.mod
```

---

## Security Constraints

Your container runs with these restrictions (you cannot change them):
- Read-only root filesystem — write only to `/tmp`
- No new privileges
- All Linux capabilities dropped
- 512 MB memory limit
- 1 CPU core limit
- 128 process limit
- Isolated network (`bench-net`) — no internet access

Your server must work within these constraints. Use `/tmp` for any temporary files.

---

## Scoring

Your submission is scored on three dimensions over a 4.5-minute benchmark:

| Component | Weight | Description |
|---|---|---|
| Throughput | 40% | Orders processed per second vs target (50,000 TPS) |
| Latency | 40% | P99 response time vs threshold (1000ms) |
| Correctness | 20% | Fill accuracy vs reference engine |

**Formula:** `finalScore = 0.40 × throughput + 0.40 × latency + 0.20 × correctness`

A submission is **disqualified** if correctness score < 30%.

---

## Tips for a High Score

1. **Latency matters as much as throughput** — a fast but incorrect server scores the same as a slow correct one. Aim for both.
2. **Handle all order types** — the bot fleet sends LIMIT, MARKET, and CANCEL orders. Returning 405 on any of them tanks your score.
3. **`/health` must respond instantly** — the sandbox waits up to 30 seconds for a 200 response before marking your submission FAILED.
4. **Write to `/tmp` only** — the filesystem is read-only. If your engine needs temp storage, use `/tmp`.
5. **Don't crash under load** — the benchmark ramps to 500 concurrent bots. Your server must stay alive for the full 4.5 minutes.
