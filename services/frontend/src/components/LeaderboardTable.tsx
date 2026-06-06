import type { LeaderboardEntry } from '../types/api';
import { StatusBadge } from './StatusBadge';

interface LeaderboardTableProps {
  entries: LeaderboardEntry[];
  loading: boolean;
}

export function LeaderboardTable({ entries, loading }: LeaderboardTableProps) {
  return (
    <div className="table-shell" id="leaderboard-table">
      <table className="leaderboard-table">
        <thead>
          <tr>
            <th>Rank</th>
            <th>Team</th>
            <th>Score</th>
            <th>TPS</th>
            <th>P99</th>
            <th>Error Rate</th>
            <th>Correctness</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {loading ? (
            <tr>
              <td colSpan={8} className="empty-row">
                <span className="loading-pulse">Loading leaderboard…</span>
              </td>
            </tr>
          ) : entries.length === 0 ? (
            <tr>
              <td colSpan={8} className="empty-row">
                No leaderboard entries yet.
              </td>
            </tr>
          ) : (
            entries.map((entry) => (
              <tr key={`${entry.teamName}-${entry.rank}`} className="leaderboard-row">
                <td className="rank-cell">
                  <span className={entry.rank <= 3 ? `rank-top rank-${entry.rank}` : ''}>
                    {entry.rank}
                  </span>
                </td>
                <td className="team-cell">{entry.teamName}</td>
                <td>{entry.finalScore.toFixed(2)}</td>
                <td>{entry.tps.toFixed(0)}</td>
                <td>{entry.p99LatencyMs.toFixed(2)} ms</td>
                <td>{entry.errorRate.toFixed(2)}%</td>
                <td>{entry.correctnessScore.toFixed(1)}%</td>
                <td>
                  <StatusBadge status={entry.status} />
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
