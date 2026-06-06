import type { LeaderboardEntry } from '../types/api';
import { StatusBadge } from './StatusBadge';

export function LeaderboardTable({ entries }: { entries: LeaderboardEntry[] }) {
  return (
    <div className="table-shell">
      <table className="leaderboard-table">
        <thead>
          <tr>
            <th>Rank</th>
            <th>Team</th>
            <th>Final</th>
            <th>TPS</th>
            <th>P99</th>
            <th>Error</th>
            <th>Correctness</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {entries.length === 0 ? (
            <tr>
              <td colSpan={8} className="empty-row">
                No leaderboard entries yet.
              </td>
            </tr>
          ) : (
            entries.map((entry) => (
              <tr key={`${entry.teamName}-${entry.rank}`}>
                <td>{entry.rank}</td>
                <td>{entry.teamName}</td>
                <td>{entry.finalScore.toFixed(2)}</td>
                <td>{entry.tps.toFixed(2)}</td>
                <td>{entry.p99LatencyMs.toFixed(2)}</td>
                <td>{entry.errorRate.toFixed(2)}%</td>
                <td>{entry.correctnessScore.toFixed(2)}%</td>
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
