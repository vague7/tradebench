package bot

import benchtypes "github.com/bench/shared/types"

type Scenario struct {
	Name  string
	Event benchtypes.BotEvent
}

func AdversarialScenarios(submissionID string) []Scenario {
	return []Scenario{
		{
			Name: "simultaneous-crossing-orders",
			Event: benchtypes.BotEvent{
				SubmissionID: submissionID,
				OrderType:    benchtypes.OrderTypeLimit,
				ExpectedFill: benchtypes.Fill{Price: 100, Quantity: 10, Side: "BUY"},
			},
		},
		{
			Name: "rapid-cancel-replace",
			Event: benchtypes.BotEvent{
				SubmissionID: submissionID,
				OrderType:    benchtypes.OrderTypeCancel,
				ExpectedFill: benchtypes.Fill{Price: 0, Quantity: 0, Side: "SELL"},
			},
		},
	}
}
