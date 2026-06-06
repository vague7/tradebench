package types

type SubmissionStatus string

const (
	StatusUploaded     SubmissionStatus = "UPLOADED"
	StatusBuilding     SubmissionStatus = "BUILDING"
	StatusRunning      SubmissionStatus = "RUNNING"
	StatusBenchmarking SubmissionStatus = "BENCHMARKING"
	StatusScored       SubmissionStatus = "SCORED"
	StatusFailed       SubmissionStatus = "FAILED"
)

type Submission struct {
	ID             string           `json:"id"`
	TeamName       string           `json:"teamName"`
	Status         SubmissionStatus `json:"status"`
	ErrorMessage   string           `json:"errorMessage,omitempty"`
	UploadedAt     string           `json:"uploadedAt"`
	BenchmarkStart string           `json:"benchmarkStart,omitempty"`
	BenchmarkEnd   string           `json:"benchmarkEnd,omitempty"`
}
