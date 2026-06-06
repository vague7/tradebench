package store

import (
	"sync"

	benchtypes "github.com/bench/shared/types"
)

type RedisClient struct {
	mu              sync.RWMutex
	jobStream       []map[string]string
	latestSnapshots map[string]benchtypes.MetricSnapshot
	leaderboard     []benchtypes.LeaderboardEntry
}

func NewRedisClient() *RedisClient {
	return &RedisClient{
		latestSnapshots: make(map[string]benchtypes.MetricSnapshot),
	}
}

func (r *RedisClient) EnqueueJob(fields map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	copied := make(map[string]string, len(fields))
	for key, value := range fields {
		copied[key] = value
	}
	r.jobStream = append(r.jobStream, copied)
}

func (r *RedisClient) SetLatestSnapshot(submissionID string, snapshot benchtypes.MetricSnapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.latestSnapshots[submissionID] = snapshot
}

func (r *RedisClient) GetLatestSnapshot(submissionID string) (benchtypes.MetricSnapshot, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot, ok := r.latestSnapshots[submissionID]
	return snapshot, ok
}

func (r *RedisClient) SetLeaderboard(entries []benchtypes.LeaderboardEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.leaderboard = append([]benchtypes.LeaderboardEntry(nil), entries...)
}

func (r *RedisClient) GetLeaderboard() []benchtypes.LeaderboardEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return append([]benchtypes.LeaderboardEntry(nil), r.leaderboard...)
}
