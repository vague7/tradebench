package store

import (
	"sort"
	"sync"
	"time"

	benchtypes "github.com/bench/shared/types"
)

type SubmissionRecord struct {
	ID             string
	TeamName       string
	Status         benchtypes.SubmissionStatus
	ErrorMessage   string
	UploadedAt     time.Time
	BenchmarkStart *time.Time
	BenchmarkEnd   *time.Time
	FinalScore     *benchtypes.Score
	Snapshot       *benchtypes.MetricSnapshot
}

type PostgresStore struct {
	mu          sync.RWMutex
	submissions map[string]*SubmissionRecord
}

func NewPostgresStore() *PostgresStore {
	return &PostgresStore{submissions: make(map[string]*SubmissionRecord)}
}

func (s *PostgresStore) CreateSubmission(id, teamName string, uploadedAt time.Time) SubmissionRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := &SubmissionRecord{
		ID:         id,
		TeamName:   teamName,
		Status:     benchtypes.StatusUploaded,
		UploadedAt: uploadedAt,
	}
	s.submissions[id] = record
	return *record
}

func (s *PostgresStore) GetSubmission(id string) (SubmissionRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.submissions[id]
	if !ok {
		return SubmissionRecord{}, false
	}
	return *record, true
}

func (s *PostgresStore) UpdateStatus(id string, status benchtypes.SubmissionStatus, message string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.submissions[id]
	if !ok {
		return false
	}
	record.Status = status
	record.ErrorMessage = message
	return true
}

func (s *PostgresStore) SetBenchmarkStart(id string, startedAt time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.submissions[id]
	if !ok {
		return false
	}
	record.BenchmarkStart = &startedAt
	record.Status = benchtypes.StatusBenchmarking
	return true
}

func (s *PostgresStore) SetBenchmarkEnd(id string, endedAt time.Time, status benchtypes.SubmissionStatus) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.submissions[id]
	if !ok {
		return false
	}
	record.BenchmarkEnd = &endedAt
	record.Status = status
	return true
}

func (s *PostgresStore) SaveResults(id string, snapshot benchtypes.MetricSnapshot, score benchtypes.Score) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.submissions[id]
	if !ok {
		return false
	}
	record.Snapshot = &snapshot
	record.FinalScore = &score
	return true
}

func (s *PostgresStore) ListLeaderboard() []benchtypes.LeaderboardEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]benchtypes.LeaderboardEntry, 0, len(s.submissions))
	for _, record := range s.submissions {
		entry := benchtypes.LeaderboardEntry{
			TeamName:         record.TeamName,
			Status:           record.Status,
			FinalScore:       0,
			TPS:              0,
			P99LatencyMs:     0,
			ErrorRate:        0,
			CorrectnessScore: 0,
		}
		if record.FinalScore != nil {
			entry.FinalScore = record.FinalScore.FinalScore * 100
			entry.CorrectnessScore = record.FinalScore.CorrectnessScore * 100
		}
		if record.Snapshot != nil {
			entry.TPS = record.Snapshot.TPS
			entry.P99LatencyMs = record.Snapshot.P99LatencyMs
			denominator := record.Snapshot.SuccessCount + record.Snapshot.FailureCount + record.Snapshot.TimeoutCount
			if denominator > 0 {
				entry.ErrorRate = float64(record.Snapshot.FailureCount+record.Snapshot.TimeoutCount) / float64(denominator) * 100
			}
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].FinalScore == entries[j].FinalScore {
			if entries[i].CorrectnessScore == entries[j].CorrectnessScore {
				return entries[i].P99LatencyMs < entries[j].P99LatencyMs
			}
			return entries[i].CorrectnessScore > entries[j].CorrectnessScore
		}
		return entries[i].FinalScore > entries[j].FinalScore
	})

	for i := range entries {
		entries[i].Rank = i + 1
	}
	return entries
}
