package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
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
	OrderID string      `json:"orderId"`
	Status  string      `json:"status"`
	Fill    *types.Fill `json:"fill,omitempty"`
}

// Bot represents a single synthetic trading bot that fires orders at a contestant container.
type Bot struct {
	BotID        string
	SubmissionID string
	targetURL    string
	timeoutMs    int
	client       *http.Client
	gen          *Generator
	emitCh       chan<- types.BotEvent
	openOrderIDs []string
}

// NewBot creates a new Bot instance.
// emitCh is a buffered channel into which the bot pushes completed BotEvent records.
// The streamer drains this channel.
func NewBot(id, submissionID, targetURL string, timeoutMs int, emitCh chan<- types.BotEvent) *Bot {
	return &Bot{
		BotID:        id,
		SubmissionID: submissionID,
		targetURL:    targetURL,
		timeoutMs:    timeoutMs,
		// Client has no global timeout — we use context.WithTimeout per request
		// so that context cancellation also aborts in-flight requests immediately.
		client: &http.Client{},
		gen:    NewGenerator(time.Now().UnixNano() ^ int64(len(id))),
		emitCh: emitCh,
	}
}

// Run is the bot's main loop. It fires orders as fast as the contestant responds.
// The concurrency (number of simultaneous bots) is the throttle mechanism, not per-bot sleep.
// It exits cleanly when ctx is cancelled.
func (b *Bot) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		b.executeOnce(ctx)
	}
}

// executeOnce performs a single order-fire-record cycle.
func (b *Bot) executeOnce(ctx context.Context) {
	order := b.gen.Next(b.openOrderIDs)

	// Build and fire HTTP request with per-request timeout
	reqCtx, reqCancel := context.WithTimeout(ctx, time.Duration(b.timeoutMs)*time.Millisecond)
	defer reqCancel()

	var req *http.Request
	var reqErr error

	switch order.Type {
	case types.OrderTypeCancel:
		url := fmt.Sprintf("%s/order/%s", b.targetURL, order.OrderIDToCancel)
		req, reqErr = http.NewRequestWithContext(reqCtx, http.MethodDelete, url, nil)
	default:
		// LIMIT or MARKET: POST /order with JSON body
		payload := orderPayload{
			Type:     string(order.Type),
			Side:     order.Side,
			Price:    order.Price,
			Quantity: order.Quantity,
		}
		body, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			slog.Error("bot: marshal order payload",
				"submissionId", b.SubmissionID,
				"botId", b.BotID,
				"err", marshalErr,
			)
			return
		}
		url := fmt.Sprintf("%s/order", b.targetURL)
		req, reqErr = http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
		if req != nil {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	if reqErr != nil {
		slog.Error("bot: create request",
			"submissionId", b.SubmissionID,
			"botId", b.BotID,
			"err", reqErr,
		)
		return
	}

	sentAt := time.Now()
	resp, httpErr := b.client.Do(req)
	ackedAt := time.Now()

	var httpStatus int
	var actualFill types.Fill
	var respOrderID string

	if httpErr != nil {
		// Timeout or network error: httpStatus = 0
		httpStatus = 0
	} else {
		defer resp.Body.Close()
		httpStatus = resp.StatusCode

		if httpStatus == http.StatusOK {
			var respBody orderResponse
			if decErr := json.NewDecoder(resp.Body).Decode(&respBody); decErr == nil {
				respOrderID = respBody.OrderID

				// Parse ActualFill from response
				if respBody.Fill != nil {
					actualFill = *respBody.Fill
				}

				// Track open order IDs
				if order.Type != types.OrderTypeCancel && respBody.OrderID != "" {
					b.openOrderIDs = append(b.openOrderIDs, respBody.OrderID)
				}

				// For successful cancels, remove the cancelled order from open orders
				if order.Type == types.OrderTypeCancel {
					b.removeOpenOrder(order.OrderIDToCancel)
				}
			}
		}
	}

	// Use response orderId if available, otherwise generate a UUID
	orderID := respOrderID
	if orderID == "" {
		orderID = uuid.NewString()
	}

	event := types.BotEvent{
		SubmissionID: b.SubmissionID,
		BotID:        b.BotID,
		OrderID:      orderID,
		OrderType:    order.Type,
		SentAt:       sentAt,
		AckedAt:      ackedAt,
		HTTPStatus:   httpStatus,
		ExpectedFill: types.Fill{}, // Day 4: zero-value; correctness validator fills on Day 5
		ActualFill:   actualFill,
	}

	// Non-blocking push onto emitCh. If channel is full, drop event.
	select {
	case b.emitCh <- event:
	default:
		slog.Warn("emit channel full, event dropped",
			"botId", b.BotID,
			"submissionId", b.SubmissionID,
		)
	}
}

// removeOpenOrder removes an order ID from the bot's open order list.
func (b *Bot) removeOpenOrder(orderID string) {
	for i, id := range b.openOrderIDs {
		if id == orderID {
			b.openOrderIDs[i] = b.openOrderIDs[len(b.openOrderIDs)-1]
			b.openOrderIDs = b.openOrderIDs[:len(b.openOrderIDs)-1]
			return
		}
	}
}
