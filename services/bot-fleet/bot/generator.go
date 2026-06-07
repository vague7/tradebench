package bot

import (
	"math/rand"
	"sync"
	"time"

	"github.com/bench/shared/types"
)

// OrderRequest represents a generated order for the bot to fire at the contestant container.
// Fields match PRD FR-3.1 order generation specification.
type OrderRequest struct {
	OrderType types.OrderType
	Side      string  // "BUY" | "SELL" | "" (empty for MARKET and CANCEL)
	Price     float64 // 0 for MARKET and CANCEL
	Quantity  float64 // 0 for CANCEL
}

// goroutine-safe package-level RNG, protected by mutex.
var (
	rngMu  sync.Mutex
	rngSrc *rand.Rand
)

func init() {
	rngSrc = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// GenerateOrder produces a random order following the distribution in PRD FR-3.1:
//   - [0.00, 0.35) → LIMIT BUY:  price = midPrice * (1 - rand in [0, 0.01)), qty in [1, 100]
//   - [0.35, 0.70) → LIMIT SELL: price = midPrice * (1 + rand in [0, 0.01)), qty in [1, 100]
//   - [0.70, 0.90) → MARKET:     price = 0, qty in [1, 100]
//   - [0.90, 1.00) → CANCEL:     price = 0, qty = 0 (falls back to MARKET if no open orders)
func GenerateOrder(midPrice float64, openOrderIDs []string) OrderRequest {
	rngMu.Lock()
	roll := rngSrc.Float64()
	priceDelta := rngSrc.Float64() * 0.01
	qty := 1.0 + rngSrc.Float64()*99.0
	rngMu.Unlock()

	switch {
	case roll < 0.35:
		return OrderRequest{
			OrderType: types.OrderTypeLimit,
			Side:      "BUY",
			Price:     midPrice * (1 - priceDelta),
			Quantity:  qty,
		}
	case roll < 0.70:
		return OrderRequest{
			OrderType: types.OrderTypeLimit,
			Side:      "SELL",
			Price:     midPrice * (1 + priceDelta),
			Quantity:  qty,
		}
	case roll < 0.90:
		return OrderRequest{
			OrderType: types.OrderTypeMarket,
			Side:      "",
			Price:     0,
			Quantity:  qty,
		}
	default:
		// CANCEL: fall back to MARKET if no open orders exist
		if len(openOrderIDs) == 0 {
			return OrderRequest{
				OrderType: types.OrderTypeMarket,
				Side:      "",
				Price:     0,
				Quantity:  qty,
			}
		}
		return OrderRequest{
			OrderType: types.OrderTypeCancel,
			Side:      "",
			Price:     0,
			Quantity:  0,
		}
	}
}

// --- Backward-compatible types for fleet/coordinator.go (Day 2+ will refactor) ---

// Order is a legacy stub type used by the existing fleet coordinator.
// Day 2: replace with OrderRequest once coordinator is rewritten.
type Order struct {
	Type         types.OrderType
	ExpectedFill types.Fill
}

// Generator is a legacy stub type used by the existing fleet coordinator.
// Day 2: replace with GenerateOrder function.
type Generator struct{}

// NewGenerator creates a new legacy Generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// Next generates an order using the legacy interface.
// Day 2: migrate callers to GenerateOrder.
func (g *Generator) Next(rng *rand.Rand) Order {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}

	roll := rng.Intn(100)
	switch {
	case roll < 35:
		return Order{Type: types.OrderTypeLimit, ExpectedFill: types.Fill{Price: 99.5, Quantity: 1, Side: "BUY"}}
	case roll < 70:
		return Order{Type: types.OrderTypeLimit, ExpectedFill: types.Fill{Price: 100.5, Quantity: 1, Side: "SELL"}}
	case roll < 90:
		return Order{Type: types.OrderTypeMarket, ExpectedFill: types.Fill{Price: 0, Quantity: 1, Side: "BUY"}}
	default:
		return Order{Type: types.OrderTypeCancel, ExpectedFill: types.Fill{Price: 0, Quantity: 0, Side: "BUY"}}
	}
}
