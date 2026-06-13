package correctness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"log/slog"

	"github.com/bench/shared/types"
)

// Validator compares actual fills from contestant responses against
// expected fills from the reference matching engine.
type Validator struct {
	refEnginePath string
}

// NewValidator creates a new correctness validator.
// refEnginePath is the filesystem path to the compiled C++ reference engine binary.
func NewValidator(refEnginePath string) *Validator {
	return &Validator{
		refEnginePath: refEnginePath,
	}
}

// Validate runs the reference engine against the given events and computes
// a correctness score on the 0–100 scale.
func (v *Validator) Validate(ctx context.Context, events []types.BotEvent) (float64, error) {
	// Step 1: Filter to validatable events
	var validatable []types.BotEvent
	for _, e := range events {
		if e.OrderType == types.OrderTypeCancel {
			continue
		}
		if e.HTTPStatus != 200 {
			continue
		}
		if e.ActualFill.Price == 0 && e.ActualFill.Quantity == 0 {
			continue
		}
		validatable = append(validatable, e)
	}

	if len(validatable) == 0 {
		return 100.0, nil
	}

	// Step 2: Build the input for the reference engine
	// TODO: plumb order parameters explicitly through BotEvent in a future sprint for exact reference engine input.
	var inputBuf bytes.Buffer
	for _, e := range validatable {
		price := 0.0
		quantity := 1.0
		side := e.ActualFill.Side

		if e.OrderType == types.OrderTypeLimit {
			if e.ActualFill.Price > 0 {
				price = e.ActualFill.Price
			} else {
				// Fall back to typical mid-price adjusted by +/- 1%
				if side == "BUY" {
					price = 99.0
				} else {
					price = 101.0
				}
			}

			if e.ActualFill.Quantity > 0 {
				quantity = e.ActualFill.Quantity
			}
		}

		type inputOrder struct {
			OrderId   string  `json:"orderId"`
			OrderType string  `json:"orderType"`
			Side      string  `json:"side"`
			Price     float64 `json:"price"`
			Quantity  float64 `json:"quantity"`
		}

		order := inputOrder{
			OrderId:   e.OrderID,
			OrderType: string(e.OrderType),
			Side:      side,
			Price:     price,
			Quantity:  quantity,
		}

		b, err := json.Marshal(order)
		if err != nil {
			return 0.0, fmt.Errorf("context: failed to marshal input order: %w", err)
		}
		inputBuf.Write(b)
		inputBuf.WriteByte('\n')
	}

	// Step 3: Invoke the reference binary
	cmd := exec.CommandContext(ctx, v.refEnginePath)
	cmd.Stdin = bytes.NewReader(inputBuf.Bytes())
	
	outBytes, err := cmd.Output()
	if err != nil {
		return 0.0, fmt.Errorf("context: correctness: ref-engine exited with error: %w", err)
	}

	// Step 4: Parse the reference engine's output
	outLines := strings.Split(strings.TrimSpace(string(outBytes)), "\n")
	// If outBytes is empty, outLines might contain one empty string, need to handle that.
	if len(outBytes) == 0 {
		outLines = []string{}
	}

	if len(outLines) != len(validatable) {
		slog.Warn("ref-engine produced mismatched output line count", "expected", len(validatable), "got", len(outLines))
		return 0.0, fmt.Errorf("context: correctness: expected %d output lines from ref-engine, got %d", len(validatable), len(outLines))
	}

	var expectedFills []types.Fill
	for i, line := range outLines {
		if line == "" {
			continue // Should not happen with well-formed output
		}
		var fill types.Fill
		if err := json.Unmarshal([]byte(line), &fill); err != nil {
			return 0.0, fmt.Errorf("context: correctness: failed to parse ref-engine output line %d: %w", i, err)
		}
		expectedFills = append(expectedFills, fill)
	}

	// Step 5: Compare actual vs expected fills
	var correctOrders int
	var totalValidated = len(validatable)

	for i, actualEvent := range validatable {
		actual := actualEvent.ActualFill
		expected := expectedFills[i]

		priceMatch := math.Abs(actual.Price-expected.Price) < 0.01
		qtyMatch := math.Abs(actual.Quantity-expected.Quantity) < 0.01
		sideMatch := actual.Side == expected.Side

		isActualZero := actual.Price == 0 && actual.Quantity == 0 && actual.Side == ""
		isExpectedZero := expected.Price == 0 && expected.Quantity == 0 && expected.Side == ""

		if isExpectedZero {
			if isActualZero {
				correctOrders++
			}
		} else {
			if priceMatch && qtyMatch && sideMatch {
				correctOrders++
			}
		}
	}

	// Step 6: Compute and return the score
	if totalValidated == 0 {
		return 100.0, nil
	}

	correctnessScore := (float64(correctOrders) / float64(totalValidated)) * 100.0
	if math.IsNaN(correctnessScore) {
		return 0.0, nil
	}

	if correctnessScore < 0 {
		correctnessScore = 0.0
	} else if correctnessScore > 100 {
		correctnessScore = 100.0
	}

	submissionId := ""
	if len(events) > 0 {
		submissionId = events[0].SubmissionID
	}

	slog.Info("correctness validation complete",
		"submissionId", submissionId,
		"totalValidated", totalValidated,
		"correctOrders", correctOrders,
		"score", correctnessScore,
	)

	return correctnessScore, nil
}
