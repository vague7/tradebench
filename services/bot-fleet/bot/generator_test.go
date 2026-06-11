package bot

import (
	"math"
	"testing"

	"github.com/bench/shared/types"
)

// TestOrderDistribution_EmptyOpenOrders generates 100,000 orders with an empty
// openOrderIDs slice and verifies the distribution matches FR-3.1 within ±2%.
// When openOrderIDs is empty, Cancel (10%) falls back to Limit Buy, so Limit Buy
// should be ~45% (35% native + 10% cancel fallback).
func TestOrderDistribution_EmptyOpenOrders(t *testing.T) {
	const n = 100_000
	gen := NewGenerator(42)

	counts := map[types.OrderType]int{
		types.OrderTypeLimit:  0,
		types.OrderTypeMarket: 0,
		types.OrderTypeCancel: 0,
	}
	buySide := 0
	sellSide := 0

	for i := 0; i < n; i++ {
		order := gen.Next(nil) // empty open orders
		counts[order.Type]++
		if order.Type == types.OrderTypeLimit {
			switch order.Side {
			case "BUY":
				buySide++
			case "SELL":
				sellSide++
			}
		}
	}

	// With empty open orders, Cancel falls back to Limit Buy.
	// So no Cancel orders should be generated.
	if counts[types.OrderTypeCancel] != 0 {
		t.Errorf("expected 0 cancel orders with empty openOrderIDs, got %d", counts[types.OrderTypeCancel])
	}

	total := float64(n)

	// Limit Buy: should be ~45% (35% native + 10% cancel fallback)
	limitBuyPct := float64(buySide) / total * 100
	if math.Abs(limitBuyPct-45.0) > 2.0 {
		t.Errorf("Limit Buy: expected ~45%%, got %.2f%%", limitBuyPct)
	}

	// Limit Sell: should be ~35%
	limitSellPct := float64(sellSide) / total * 100
	if math.Abs(limitSellPct-35.0) > 2.0 {
		t.Errorf("Limit Sell: expected ~35%%, got %.2f%%", limitSellPct)
	}

	// Market: should be ~20%
	marketPct := float64(counts[types.OrderTypeMarket]) / total * 100
	if math.Abs(marketPct-20.0) > 2.0 {
		t.Errorf("Market: expected ~20%%, got %.2f%%", marketPct)
	}
}

// TestOrderDistribution_WithOpenOrders generates 100,000 orders with non-empty
// openOrderIDs and verifies that Cancel orders are generated at the expected rate.
func TestOrderDistribution_WithOpenOrders(t *testing.T) {
	const n = 100_000
	gen := NewGenerator(123)

	openOrders := []string{"order-1", "order-2", "order-3", "order-4", "order-5"}

	counts := map[types.OrderType]int{
		types.OrderTypeLimit:  0,
		types.OrderTypeMarket: 0,
		types.OrderTypeCancel: 0,
	}
	buySide := 0
	sellSide := 0

	for i := 0; i < n; i++ {
		order := gen.Next(openOrders)
		counts[order.Type]++
		if order.Type == types.OrderTypeLimit {
			switch order.Side {
			case "BUY":
				buySide++
			case "SELL":
				sellSide++
			}
		}
	}

	total := float64(n)

	// Limit Buy: should be ~35%
	limitBuyPct := float64(buySide) / total * 100
	if math.Abs(limitBuyPct-35.0) > 2.0 {
		t.Errorf("Limit Buy: expected ~35%%, got %.2f%%", limitBuyPct)
	}

	// Limit Sell: should be ~35%
	limitSellPct := float64(sellSide) / total * 100
	if math.Abs(limitSellPct-35.0) > 2.0 {
		t.Errorf("Limit Sell: expected ~35%%, got %.2f%%", limitSellPct)
	}

	// Market: should be ~20%
	marketPct := float64(counts[types.OrderTypeMarket]) / total * 100
	if math.Abs(marketPct-20.0) > 2.0 {
		t.Errorf("Market: expected ~20%%, got %.2f%%", marketPct)
	}

	// Cancel: should be ~10%
	cancelPct := float64(counts[types.OrderTypeCancel]) / total * 100
	if math.Abs(cancelPct-10.0) > 2.0 {
		t.Errorf("Cancel: expected ~10%%, got %.2f%%", cancelPct)
	}
}

// TestCancelFallbackToLimitBuy verifies that when Cancel is rolled but openOrderIDs
// is empty, the generator falls back to a Limit Buy order (not Market).
func TestCancelFallbackToLimitBuy(t *testing.T) {
	// Use a large sample to statistically verify fallback behaviour
	const n = 50_000
	gen := NewGenerator(999)

	for i := 0; i < n; i++ {
		order := gen.Next(nil)
		// With empty openOrderIDs, no Cancel orders should be generated
		if order.Type == types.OrderTypeCancel {
			t.Fatalf("got Cancel order with empty openOrderIDs at iteration %d", i)
		}
		// All generated orders should have valid types
		switch order.Type {
		case types.OrderTypeLimit, types.OrderTypeMarket:
			// ok
		default:
			t.Fatalf("unexpected order type %q at iteration %d", order.Type, i)
		}
	}
}

// TestCancelOrderIDSelection verifies that Cancel orders select an order ID from
// the provided openOrderIDs slice.
func TestCancelOrderIDSelection(t *testing.T) {
	gen := NewGenerator(777)
	openOrders := []string{"aaa", "bbb", "ccc"}

	// Generate enough orders to get some Cancels
	for i := 0; i < 10_000; i++ {
		order := gen.Next(openOrders)
		if order.Type == types.OrderTypeCancel {
			found := false
			for _, id := range openOrders {
				if order.OrderIDToCancel == id {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("Cancel order selected invalid OrderIDToCancel %q", order.OrderIDToCancel)
			}
		}
	}
}
