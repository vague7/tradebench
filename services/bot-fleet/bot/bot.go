package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/bench/shared/types"
	"github.com/google/uuid"
)

// orderPayload is the JSON body sent to the contestant container for LIMIT/MARKET orders.
// Matches PRD Section 2.3.3 contestant container request body.
type orderPayload struct {
	Type     string  `json:"type"`
	Side     string  `json:"side"`
	Price    float64 `json:"price"`
	Quantity float64 `json:"qty"`
}

// orderResponse is the JSON body received from the contestant container.
// Matches PRD Section 2.3.3 contestant container response body.
type orderResponse struct {
	OrderID string     `json:"orderId"`
	Status  string     `json:"status"`
	Fill    *types.Fill `json:"fill,omitempty"`
}

// RunBot executes a single bot invocation: generates an order, fires it as a real HTTP
// request to the contestant container, records timing, and returns a BotEvent.
//
// This function is designed to be called in a loop from a goroutine. The coordinator
// (Day 2+) will spawn N of these concurrently.
func RunBot(ctx context.Context, submissionID, targetBaseURL string, timeoutMs int, openOrderIDs *[]string, mu *sync.Mutex) (types.BotEvent, error) {
	botID := uuid.NewString()
	orderID := uuid.NewString()

	// Snapshot openOrderIDs for reading
	mu.Lock()
	snapshot := make([]string, len(*openOrderIDs))
	copy(snapshot, *openOrderIDs)
	mu.Unlock()

	// Day 1: hardcoded midPrice of 100.0 (real mid-price will come from orderbook endpoint in later days)
	orderReq := GenerateOrder(100.0, snapshot)

	// Build HTTP request based on order type
	reqCtx, reqCancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer reqCancel()

	var req *http.Request
	var reqErr error

	switch orderReq.OrderType {
	case types.OrderTypeCancel:
		// Pick a random open order ID for cancellation
		rngMu.Lock()
		idx := rngSrc.Intn(len(snapshot))
		rngMu.Unlock()
		cancelID := snapshot[idx]
		url := fmt.Sprintf("%s/order/%s", targetBaseURL, cancelID)
		req, reqErr = http.NewRequestWithContext(reqCtx, http.MethodDelete, url, nil)
	default:
		// LIMIT or MARKET: POST /order with JSON body
		payload := orderPayload{
			Type:     string(orderReq.OrderType),
			Side:     orderReq.Side,
			Price:    orderReq.Price,
			Quantity: orderReq.Quantity,
		}
		body, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return types.BotEvent{}, fmt.Errorf("bot: marshal order payload for %s: %w", submissionID, marshalErr)
		}
		url := fmt.Sprintf("%s/order", targetBaseURL)
		req, reqErr = http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
		if req != nil {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	if reqErr != nil {
		return types.BotEvent{}, fmt.Errorf("bot: create request for %s: %w", submissionID, reqErr)
	}

	// Fire the request with a dedicated client (not DefaultClient — it has no timeout)
	client := &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond}

	sentAt := time.Now()
	resp, httpErr := client.Do(req)
	ackedAt := time.Now()

	var httpStatus int
	var actualFill types.Fill

	if httpErr != nil {
		// Timeout or network error: httpStatus = 0
		httpStatus = 0
		slog.Debug("bot: request error",
			"submissionId", submissionID,
			"botId", botID,
			"orderType", orderReq.OrderType,
			"err", httpErr,
		)
	} else {
		defer resp.Body.Close()
		httpStatus = resp.StatusCode

		if httpStatus == http.StatusOK {
			var respBody orderResponse
			if decErr := json.NewDecoder(resp.Body).Decode(&respBody); decErr == nil {
				// Parse ActualFill from response
				if respBody.Fill != nil {
					actualFill = *respBody.Fill
				}
				// If order accepted and non-CANCEL, track the open order ID
				if orderReq.OrderType != types.OrderTypeCancel && respBody.OrderID != "" {
					mu.Lock()
					*openOrderIDs = append(*openOrderIDs, respBody.OrderID)
					mu.Unlock()
				}
			}
		}
	}

	return types.BotEvent{
		SubmissionID: submissionID,
		BotID:        botID,
		OrderID:      orderID,
		OrderType:    orderReq.OrderType,
		SentAt:       sentAt,
		AckedAt:      ackedAt,
		HTTPStatus:   httpStatus,
		ExpectedFill: types.Fill{}, // Day 1: reference engine not yet wired
		ActualFill:   actualFill,
	}, nil
}

// --- Backward-compatible types for fleet/coordinator.go (Day 2+ will refactor) ---

// Streamer is a legacy interface used by the existing fleet coordinator.
// Day 2: wire gRPC stream here via emit/streamer.go
type Streamer interface {
	Send(context.Context, types.BotEvent) error
}

// Runner is a legacy stub type used by the existing fleet coordinator.
// Day 2: replace with RunBot-based coordinator.
type Runner struct {
	Generator *Generator
	Streamer  Streamer
}

// NewRunner creates a new legacy Runner.
func NewRunner(generator *Generator, streamer Streamer) *Runner {
	return &Runner{Generator: generator, Streamer: streamer}
}

// Run executes bot iterations using the legacy interface.
// Day 2: migrate to RunBot-based goroutine pool.
func (r *Runner) Run(ctx context.Context, submissionID, botID string, iterations int) error {
	if iterations <= 0 {
		iterations = 1
	}
	for i := 0; i < iterations; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		order := r.Generator.Next(rand.New(rand.NewSource(time.Now().UnixNano() + int64(i))))
		event := types.BotEvent{
			SubmissionID: submissionID,
			BotID:        botID,
			OrderID:      fmt.Sprintf("%s-%d", botID, i),
			OrderType:    order.Type,
			SentAt:       time.Now().UTC(),
			AckedAt:      time.Now().UTC(),
			HTTPStatus:   200,
			ExpectedFill: order.ExpectedFill,
			ActualFill:   order.ExpectedFill,
		}
		if err := r.Streamer.Send(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
