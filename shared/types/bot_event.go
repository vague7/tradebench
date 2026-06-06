package types

import "time"

type Fill struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Side     string  `json:"side"`
}

type OrderType string

const (
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeCancel OrderType = "CANCEL"
)

type BotEvent struct {
	SubmissionID string    `json:"submissionId"`
	BotID        string    `json:"botId"`
	OrderID      string    `json:"orderId"`
	OrderType    OrderType `json:"orderType"`
	SentAt       time.Time `json:"sentAt"`
	AckedAt      time.Time `json:"ackedAt"`
	HTTPStatus   int       `json:"httpStatus"`
	ExpectedFill Fill      `json:"expectedFill"`
	ActualFill   Fill      `json:"actualFill"`
}
