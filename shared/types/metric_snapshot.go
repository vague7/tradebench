package types

import "time"

type MetricSnapshot struct {
	SubmissionID     string    `json:"submissionId"`
	WindowEnd        time.Time `json:"windowEnd"`
	P50LatencyMs     float64   `json:"p50LatencyMs"`
	P90LatencyMs     float64   `json:"p90LatencyMs"`
	P99LatencyMs     float64   `json:"p99LatencyMs"`
	TPS              float64   `json:"tps"`
	SuccessCount     int64     `json:"successCount"`
	FailureCount     int64     `json:"failureCount"`
	TimeoutCount     int64     `json:"timeoutCount"`
	CorrectnessScore float64   `json:"correctnessScore"`
}
