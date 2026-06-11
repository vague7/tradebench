package bot

import (
	"math/rand"

	"github.com/bench/shared/types"
)

// defaultMidPrice is used as the reference mid-price for order generation.
// Day 4: static value. A realistic price feed is a Day 5/6 enhancement.
const defaultMidPrice = 100.0

// OrderSpec captures all fields needed to construct the HTTP request body for one order.
type OrderSpec struct {
	Type            types.OrderType // LIMIT, MARKET, or CANCEL
	Side            string          // "BUY" or "SELL"; empty for cancel/market
	Price           float64         // 0 for market/cancel
	Quantity        float64         // 0 for cancel
	OrderIDToCancel string          // UUID of order to cancel; empty otherwise
}

// Generator produces orders following the PRD FR-3.1 distribution.
// Each bot has its own Generator with its own *rand.Rand to avoid global lock
// contention at 50k concurrent goroutines.
type Generator struct {
	rng      *rand.Rand
	midPrice float64
}

// NewGenerator creates a new Generator with a per-bot random source.
// Each bot passes its own unique seed (e.g. derived from bot index) so
// generators are statistically independent.
func NewGenerator(seed int64) *Generator {
	return &Generator{
		rng:      rand.New(rand.NewSource(seed)),
		midPrice: defaultMidPrice,
	}
}

// Next generates the next order according to the FR-3.1 distribution:
//   - [0.00, 0.35) → Limit Buy:  price within 1% below mid, qty 1–100
//   - [0.35, 0.70) → Limit Sell: price within 1% above mid, qty 1–100
//   - [0.70, 0.90) → Market:     no price, qty 1–100
//   - [0.90, 1.00) → Cancel:     cancel a random open order
//
// If a Cancel is rolled but openOrderIDs is empty, fall back to Limit Buy.
func (g *Generator) Next(openOrderIDs []string) OrderSpec {
	roll := g.rng.Float64()

	switch {
	case roll < 0.35:
		return g.limitBuy()
	case roll < 0.70:
		return g.limitSell()
	case roll < 0.90:
		return g.market()
	default:
		// CANCEL: fall back to Limit Buy if no open orders exist
		if len(openOrderIDs) == 0 {
			return g.limitBuy()
		}
		idx := g.rng.Intn(len(openOrderIDs))
		return OrderSpec{
			Type:            types.OrderTypeCancel,
			Side:            "",
			Price:           0,
			Quantity:        0,
			OrderIDToCancel: openOrderIDs[idx],
		}
	}
}

// limitBuy generates a Limit Buy order: price within 1% below mid-price, qty 1–100.
func (g *Generator) limitBuy() OrderSpec {
	priceDelta := g.rng.Float64() * 0.01
	qty := 1.0 + g.rng.Float64()*99.0
	return OrderSpec{
		Type:     types.OrderTypeLimit,
		Side:     "BUY",
		Price:    g.midPrice * (1 - priceDelta),
		Quantity: qty,
	}
}

// limitSell generates a Limit Sell order: price within 1% above mid-price, qty 1–100.
func (g *Generator) limitSell() OrderSpec {
	priceDelta := g.rng.Float64() * 0.01
	qty := 1.0 + g.rng.Float64()*99.0
	return OrderSpec{
		Type:     types.OrderTypeLimit,
		Side:     "SELL",
		Price:    g.midPrice * (1 + priceDelta),
		Quantity: qty,
	}
}

// market generates a Market order: no price, qty 1–100.
func (g *Generator) market() OrderSpec {
	qty := 1.0 + g.rng.Float64()*99.0
	return OrderSpec{
		Type:     types.OrderTypeMarket,
		Side:     "",
		Price:    0,
		Quantity: qty,
	}
}
