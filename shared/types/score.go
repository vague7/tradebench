package types

import "time"

type Score struct {
	SubmissionID     string    `json:"submissionId"`
	TeamName         string    `json:"teamName"`
	ThroughputScore   float64   `json:"throughputScore"`
	LatencyScore     float64   `json:"latencyScore"`
	CorrectnessScore float64   `json:"correctnessScore"`
	FinalScore       float64   `json:"finalScore"`
	IsDisqualified   bool      `json:"isDisqualified"`
	DisqualifyReason string    `json:"disqualifyReason,omitempty"`
	ComputedAt       time.Time `json:"computedAt"`
}

type LeaderboardEntry struct {
	Rank             int              `json:"rank"`
	TeamName         string           `json:"teamName"`
	FinalScore       float64          `json:"finalScore"`
	TPS              float64          `json:"tps"`
	P99LatencyMs     float64          `json:"p99LatencyMs"`
	ErrorRate        float64          `json:"errorRate"`
	CorrectnessScore float64          `json:"correctnessScore"`
	Status           SubmissionStatus `json:"status"`
	IsDisqualified   bool             `json:"isDisqualified"`
}
