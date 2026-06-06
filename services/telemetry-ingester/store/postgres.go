package store

import (
	"sync"

	benchtypes "github.com/bench/shared/types"
)

type PostgresStore struct {
	mu      sync.Mutex
	scores  []benchtypes.Score
	snaps   []benchtypes.MetricSnapshot
}

func NewPostgresStore() *PostgresStore {
	return &PostgresStore{}
}

func (s *PostgresStore) SaveSnapshot(snapshot benchtypes.MetricSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snaps = append(s.snaps, snapshot)
}

func (s *PostgresStore) SaveScore(score benchtypes.Score) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.scores = append(s.scores, score)
}

func (s *PostgresStore) LatestScore(submissionID string) (benchtypes.Score, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := len(s.scores) - 1; i >= 0; i-- {
		if s.scores[i].SubmissionID == submissionID {
			return s.scores[i], true
		}
	}
	return benchtypes.Score{}, false
}
