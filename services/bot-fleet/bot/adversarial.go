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

// SimultaneousCrossingOrders spawns 1,000 short-lived bot goroutines that each submit
// a matching buy/sell pair within 1ms of each other to expose race conditions in the
// contestant's matching engine. Both sides of each pair are submitted by different
// goroutines firing at the same instant via a sync.WaitGroup barrier.
func SimultaneousCrossingOrders(ctx context.Context, submissionID, targetURL string, timeoutMs int, emitCh chan<- types.BotEvent) {
	slog.Info("adversarial scenario injected", "scenario", "simultaneous-crossing-orders", "submissionId", submissionID)

	const pairCount = 500 // 500 pairs = 1000 goroutines
	var barrier sync.WaitGroup
	barrier.Add(pairCount * 2)

	var done sync.WaitGroup
	done.Add(pairCount * 2)

	// Hold all goroutines at the barrier, then release simultaneously
	var gate sync.WaitGroup
	gate.Add(1)

	client := &http.Client{}
	price := 100.0

	for i := 0; i < pairCount; i++ {
		// Buy side
		go func() {
			defer done.Done()
			barrier.Done()
			gate.Wait() // wait for simultaneous release

			select {
			case <-ctx.Done():
				return
			default:
			}

			fireAdversarialOrder(ctx, client, submissionID, targetURL, timeoutMs, "LIMIT", "BUY", price, 10, emitCh)
		}()

		// Sell side
		go func() {
			defer done.Done()
			barrier.Done()
			gate.Wait() // wait for simultaneous release

			select {
			case <-ctx.Done():
				return
			default:
			}

			fireAdversarialOrder(ctx, client, submissionID, targetURL, timeoutMs, "LIMIT", "SELL", price, 10, emitCh)
		}()
	}

	// Wait for all goroutines to reach the barrier, then release
	barrier.Wait()
	gate.Done()

	// Wait for all goroutines to complete
	done.Wait()
}

// RapidCancelReplace submits a limit order, immediately cancels it, and resubmits
// with the same semantic intent (new UUID, same price/quantity) within 5ms.
// Runs 100 concurrent instances to test idempotency handling.
func RapidCancelReplace(ctx context.Context, submissionID, targetURL string, timeoutMs int, emitCh chan<- types.BotEvent) {
	slog.Info("adversarial scenario injected", "scenario", "rapid-cancel-replace", "submissionId", submissionID)

	const instances = 100
	var wg sync.WaitGroup
	wg.Add(instances)

	client := &http.Client{}
	price := 100.0
	qty := 50.0

	for i := 0; i < instances; i++ {
		go func() {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			// Step 1: Submit limit order
			orderID := fireAdversarialOrder(ctx, client, submissionID, targetURL, timeoutMs, "LIMIT", "BUY", price, qty, emitCh)

			// Step 2: Immediately cancel it
			if orderID != "" {
				fireAdversarialCancel(ctx, client, submissionID, targetURL, timeoutMs, orderID, emitCh)
			}

			// Step 3: Resubmit with same price/quantity (new UUID generated server-side)
			fireAdversarialOrder(ctx, client, submissionID, targetURL, timeoutMs, "LIMIT", "BUY", price, qty, emitCh)
		}()
	}

	wg.Wait()
}

// OrderBookFlood spawns goroutines that submit 10,000 limit orders all at the same
// price level to stress the contestant's order book data structure.
func OrderBookFlood(ctx context.Context, submissionID, targetURL string, timeoutMs int, emitCh chan<- types.BotEvent) {
	slog.Info("adversarial scenario injected", "scenario", "order-book-flood", "submissionId", submissionID)

	const orderCount = 10_000
	const concurrency = 100 // submit via 100 goroutines, each doing 100 orders

	var wg sync.WaitGroup
	wg.Add(concurrency)

	client := &http.Client{}
	fixedPrice := 99.50 // fixed price level instead of mid-price generator

	perWorker := orderCount / concurrency

	for w := 0; w < concurrency; w++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				rng := rand.New(rand.NewSource(time.Now().UnixNano()))
				qty := 1.0 + rng.Float64()*99.0
				fireAdversarialOrder(ctx, client, submissionID, targetURL, timeoutMs, "LIMIT", "BUY", fixedPrice, qty, emitCh)
			}
		}()
	}

	wg.Wait()
}

// FatFinger injects an order with 10x the normal maximum quantity (1,000–10,000
// instead of the normal 1–100) to test boundary handling. This is a single order,
// not a burst.
func FatFinger(ctx context.Context, submissionID, targetURL string, timeoutMs int, emitCh chan<- types.BotEvent) {
	slog.Info("adversarial scenario injected", "scenario", "fat-finger", "submissionId", submissionID)

	select {
	case <-ctx.Done():
		return
	default:
	}

	client := &http.Client{}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	fatQty := 1000.0 + rng.Float64()*9000.0 // 1,000 to 10,000
	fireAdversarialOrder(ctx, client, submissionID, targetURL, timeoutMs, "LIMIT", "BUY", 100.0, fatQty, emitCh)
}

// StaleCancel attempts to cancel an order ID that does not exist (randomly generated UUID).
// Verifies the contestant returns a non-200 status or an appropriate error body.
// Logs the result but does not count it as a bot failure — this is an expected error path.
func StaleCancel(ctx context.Context, submissionID, targetURL string, timeoutMs int, emitCh chan<- types.BotEvent) {
	slog.Info("adversarial scenario injected", "scenario", "stale-cancel", "submissionId", submissionID)

	select {
	case <-ctx.Done():
		return
	default:
	}

	client := &http.Client{}
	fakeOrderID := uuid.NewString() // UUID that was never submitted
	fireAdversarialCancel(ctx, client, submissionID, targetURL, timeoutMs, fakeOrderID, emitCh)
}

// fireAdversarialOrder sends a single order and returns the orderId from the response.
func fireAdversarialOrder(ctx context.Context, client *http.Client, submissionID, targetURL string, timeoutMs int, orderType, side string, price, qty float64, emitCh chan<- types.BotEvent) string {
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	payload := orderPayload{
		Type:     orderType,
		Side:     side,
		Price:    price,
		Quantity: qty,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("adversarial: marshal payload", "err", err)
		return ""
	}

	url := fmt.Sprintf("%s/order", targetURL)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		slog.Error("adversarial: create request", "err", err)
		return ""
	}
	req.Header.Set("Content-Type", "application/json")

	botID := uuid.NewString()
	sentAt := time.Now()
	resp, httpErr := client.Do(req)
	ackedAt := time.Now()

	var httpStatus int
	var respOrderID string
	var actualFill types.Fill

	if httpErr != nil {
		httpStatus = 0
	} else {
		defer resp.Body.Close()
		httpStatus = resp.StatusCode
		if httpStatus == http.StatusOK {
			var respBody orderResponse
			if decErr := json.NewDecoder(resp.Body).Decode(&respBody); decErr == nil {
				respOrderID = respBody.OrderID
				if respBody.Fill != nil {
					actualFill = *respBody.Fill
				}
			}
		}
	}

	orderID := respOrderID
	if orderID == "" {
		orderID = uuid.NewString()
	}

	event := types.BotEvent{
		SubmissionID: submissionID,
		BotID:        botID,
		OrderID:      orderID,
		OrderType:    types.OrderType(orderType),
		SentAt:       sentAt,
		AckedAt:      ackedAt,
		HTTPStatus:   httpStatus,
		ExpectedFill: types.Fill{},
		ActualFill:   actualFill,
	}

	select {
	case emitCh <- event:
	default:
		slog.Warn("emit channel full, event dropped",
			"botId", botID,
			"submissionId", submissionID,
		)
	}

	return respOrderID
}

// fireAdversarialCancel sends a DELETE /order/:orderId request.
func fireAdversarialCancel(ctx context.Context, client *http.Client, submissionID, targetURL string, timeoutMs int, orderIDToCancel string, emitCh chan<- types.BotEvent) {
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	url := fmt.Sprintf("%s/order/%s", targetURL, orderIDToCancel)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodDelete, url, nil)
	if err != nil {
		slog.Error("adversarial: create cancel request", "err", err)
		return
	}

	botID := uuid.NewString()
	sentAt := time.Now()
	resp, httpErr := client.Do(req)
	ackedAt := time.Now()

	var httpStatus int
	if httpErr != nil {
		httpStatus = 0
	} else {
		defer resp.Body.Close()
		httpStatus = resp.StatusCode
	}

	event := types.BotEvent{
		SubmissionID: submissionID,
		BotID:        botID,
		OrderID:      orderIDToCancel,
		OrderType:    types.OrderTypeCancel,
		SentAt:       sentAt,
		AckedAt:      ackedAt,
		HTTPStatus:   httpStatus,
		ExpectedFill: types.Fill{},
		ActualFill:   types.Fill{},
	}

	select {
	case emitCh <- event:
	default:
		slog.Warn("emit channel full, event dropped",
			"botId", botID,
			"submissionId", submissionID,
		)
	}
}
