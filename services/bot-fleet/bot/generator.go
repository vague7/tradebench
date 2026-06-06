package bot

import "math/rand"

import benchtypes "github.com/bench/shared/types"

type Order struct {
	Type          benchtypes.OrderType
	ExpectedFill  benchtypes.Fill
}

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) Next(rng *rand.Rand) Order {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}

	roll := rng.Intn(100)
	switch {
	case roll < 35:
		return Order{Type: benchtypes.OrderTypeLimit, ExpectedFill: benchtypes.Fill{Price: 99.5, Quantity: 1, Side: "BUY"}}
	case roll < 70:
		return Order{Type: benchtypes.OrderTypeLimit, ExpectedFill: benchtypes.Fill{Price: 100.5, Quantity: 1, Side: "SELL"}}
	case roll < 90:
		return Order{Type: benchtypes.OrderTypeMarket, ExpectedFill: benchtypes.Fill{Price: 0, Quantity: 1, Side: "BUY"}}
	default:
		return Order{Type: benchtypes.OrderTypeCancel, ExpectedFill: benchtypes.Fill{Price: 0, Quantity: 0, Side: "BUY"}}
	}
}
