import { useEffect, useRef, useState } from 'react';

import type { LeaderboardEntry } from '../types/api';
import { StatusPill, DisqualifiedPill } from './StatusPill';

interface LeaderboardTableProps {
  entries: LeaderboardEntry[];
  loading: boolean;
}

function errorRateClass(rate: number): string {
  if (rate < 1) return 'sev--green';
  if (rate <= 5) return 'sev--amber';
  return 'sev--red';
}

function correctnessClass(pct: number): string {
  if (pct >= 90) return 'sev--green';
  if (pct >= 30) return 'sev--amber';
  return 'sev--red';
}

function timeAgo(ts: string | undefined): string {
  if (!ts) return '—';
  const s = Math.floor((Date.now() - new Date(ts).getTime()) / 1000);
  if (s < 5) return 'just now';
  if (s < 60) return `${s}s ago`;
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  return `${Math.floor(s / 3600)}h ago`;
}

/** Normalize 0-1 scale to 0-100 for display. Values already 0-100 pass through. */
function normCorrect(raw: number): number {
  return raw <= 1.0 ? raw * 100 : raw;
}

export function LeaderboardTable({ entries, loading }: LeaderboardTableProps) {
  const [tab, setTab] = useState<'rankings' | 'failing'>('rankings');
  const prevRef = useRef<Map<string, number>>(new Map());
  const [flashedTeams, setFlashedTeams] = useState<Set<string>>(new Set());

  const qualified = entries.filter((e) => !e.isDisqualified && e.status !== 'FAILED');
  const failing = entries.filter((e) => e.isDisqualified || e.status === 'FAILED');

  // Flash rows that changed rank
  useEffect(() => {
    const prev = prevRef.current;
    const flashed = new Set<string>();
    entries.forEach((e) => {
      const prevRank = prev.get(e.teamName);
      if (prevRank !== undefined && prevRank !== e.rank) flashed.add(e.teamName);
    });
    if (flashed.size > 0) {
      setFlashedTeams(flashed);
      const t = setTimeout(() => setFlashedTeams(new Set()), 1200);
      return () => clearTimeout(t);
    }
    const next = new Map<string, number>();
    entries.forEach((e) => next.set(e.teamName, e.rank));
    prevRef.current = next;
  }, [entries]);

  useEffect(() => {
    if (flashedTeams.size === 0) {
      const next = new Map<string, number>();
      entries.forEach((e) => next.set(e.teamName, e.rank));
      prevRef.current = next;
    }
  }, [flashedTeams, entries]);

  function rankChange(team: string, rank: number): 'up' | 'down' | null {
    const prev = prevRef.current.get(team);
    if (prev === undefined || prev === rank) return null;
    return rank < prev ? 'up' : 'down';
  }

  const showEntries = tab === 'rankings' ? qualified : failing;

  return (
    <div className="lb-container">
      {/* Tabs */}
      <div className="lb-tabs">
        <button
          className={`lb-tab${tab === 'rankings' ? ' lb-tab--active' : ''}`}
          onClick={() => setTab('rankings')}
          type="button"
        >
          Main Rankings
          {qualified.length > 0 && <span className="lb-tab-count">{qualified.length}</span>}
        </button>
        <button
          className={`lb-tab${tab === 'failing' ? ' lb-tab--active' : ''}`}
          onClick={() => setTab('failing')}
          type="button"
        >
          Failing
          {failing.length > 0 && <span className="lb-tab-count lb-tab-count--red">{failing.length}</span>}
        </button>
      </div>

      <div className="table-shell">
        <table className="lb-table">
          <thead>
            <tr>
              {tab === 'rankings' && <th>Rank</th>}
              <th>Team</th>
              <th>Score</th>
              <th>TPS</th>
              <th>P99</th>
              <th>Error %</th>
              <th>Correct %</th>
              <th>Status</th>
              {tab === 'failing' && <th>Reason</th>}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={tab === 'rankings' ? 8 : 8} className="table-empty"><span className="loading-pulse">Loading…</span></td></tr>
            ) : showEntries.length === 0 ? (
              <tr>
                <td colSpan={tab === 'rankings' ? 8 : 8} className="table-empty">
                  {tab === 'rankings'
                    ? 'No scored submissions yet.'
                    : 'No disqualified submissions.'}
                </td>
              </tr>
            ) : (
              showEntries.map((e) => {
                const change = tab === 'rankings' ? rankChange(e.teamName, e.rank) : null;
                const isFlashed = flashedTeams.has(e.teamName);

                return (
                  <tr
                    key={`${e.teamName}-${e.rank}`}
                    className={`lb-row${isFlashed ? ' row-flash' : ''}${e.isDisqualified ? ' row-dq' : ''}`}
                  >
                    {tab === 'rankings' && (
                      <td className="rank-cell">
                        <span className={e.rank <= 3 ? `rank-badge rank-${e.rank}` : ''}>
                          {e.rank}
                        </span>
                        {change === 'up' && <span className="rank-arrow rank-arrow--up" title="Moved up">▲</span>}
                        {change === 'down' && <span className="rank-arrow rank-arrow--down" title="Moved down">▼</span>}
                      </td>
                    )}
                    <td className="team-cell">{e.teamName}</td>
                    <td>
                      <div className="score-cell">
                        <span className="score-num">{e.finalScore.toFixed(2)}</span>
                        <div className="score-bar">
                          <div className="score-bar-fill" style={{ width: `${Math.min(e.finalScore, 100)}%` }} />
                        </div>
                      </div>
                    </td>
                    <td>{e.tps.toFixed(0)}</td>
                    <td className="mono">{e.p99LatencyMs.toFixed(1)}<span className="unit"> ms</span></td>
                    <td>
                      <span className={`sev-pill ${errorRateClass(e.errorRate)}`}>{e.errorRate.toFixed(2)}%</span>
                    </td>
                    <td>
                      <span className={`sev-pill ${correctnessClass(normCorrect(e.correctnessScore))}`}>{normCorrect(e.correctnessScore).toFixed(1)}%</span>
                    </td>
                    <td>
                      {e.isDisqualified ? <DisqualifiedPill /> : <StatusPill status={e.status} />}
                    </td>
                    {tab === 'failing' && (
                      <td className="reason-cell">
                        {e.status === 'FAILED' ? 'Build/Runtime Failed' : normCorrect(e.correctnessScore) < 30 ? 'Correctness < 30%' : 'Disqualified'}
                      </td>
                    )}
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
