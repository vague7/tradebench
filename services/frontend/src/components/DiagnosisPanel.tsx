import { useEffect, useRef, useState } from 'react';
import type { MetricSnapshot } from '../types/api';

interface DiagnosisPanelProps {
  snapshots: MetricSnapshot[];
  submissionId: string | null;
}

type Severity = 'critical' | 'warning' | 'ok';

interface DiagnosisCard {
  id: string;
  severity: Severity;
  title: string;
  detail: string;
}

// ── Analysis helpers ──────────────────────────────────────────────────────────

function errRate(s: MetricSnapshot): number {
  const total = s.successCount + s.failureCount + s.timeoutCount;
  return total > 0 ? (s.failureCount + s.timeoutCount) / total : 0;
}

function norm(v: number): number {
  return v <= 1.0 ? v * 100 : v;
}

// ── Rule Engine ───────────────────────────────────────────────────────────────

function diagnose(snapshots: MetricSnapshot[]): DiagnosisCard[] {
  const cards: DiagnosisCard[] = [];
  if (snapshots.length < 3) return cards;

  const first = snapshots[0];
  const last = snapshots[snapshots.length - 1];

  let maxP99 = 0, sumP99 = 0, sumErr = 0, sumCorr = 0;
  snapshots.forEach(s => {
    maxP99 = Math.max(maxP99, s.p99LatencyMs);
    sumP99 += s.p99LatencyMs;
    sumErr += errRate(s);
    sumCorr += norm(s.correctnessScore);
  });

  const avgP99 = sumP99 / snapshots.length;
  const overallErrPct = (sumErr / snapshots.length) * 100;
  const avgCorr = sumCorr / snapshots.length;
  const startErr = errRate(first);
  const endErr = errRate(last);
  const tpsStart = first.tps;
  const tpsEnd = last.tps;
  const tpsDrop = tpsStart > 0 ? (tpsStart - tpsEnd) / tpsStart : 0;
  const totalTimeouts = snapshots.reduce((a, s) => a + s.timeoutCount, 0);
  const totalFailed = snapshots.reduce((a, s) => a + s.failureCount, 0);

  // RULE 1: Cascading failure
  if (endErr > startErr + 0.1 && endErr > 0.05 && last.tps < first.tps) {
    cards.push({
      id: 'cascading-failure', severity: 'critical',
      title: 'Cascading Failure',
      detail: `Error rate ${(startErr*100).toFixed(1)}% → ${(endErr*100).toFixed(1)}% as TPS collapsed. Queue backlog overwhelmed workers.`,
    });
  }

  // RULE 2: Latency spikes
  if (maxP99 > avgP99 * 3 && maxP99 > 500) {
    cards.push({
      id: 'latency-spike', severity: maxP99 > 1500 ? 'critical' : 'warning',
      title: 'Tail Latency Spikes',
      detail: `P99 hit ${maxP99.toFixed(0)}ms vs avg ${avgP99.toFixed(0)}ms — likely GC pause or hot-path lock contention.`,
    });
  }

  // RULE 3: Timeouts
  if (totalTimeouts > 500) {
    cards.push({
      id: 'frequent-timeouts', severity: 'critical',
      title: 'High Timeout Volume',
      detail: `${totalTimeouts.toLocaleString()} requests hit gateway timeout. Matching engine too slow for incoming TPS.`,
    });
  }

  // RULE 4: Fast but wrong (race condition)
  if (avgP99 < 50 && avgCorr < 90) {
    cards.push({
      id: 'race-condition', severity: avgCorr < 50 ? 'critical' : 'warning',
      title: 'Fast but Incorrect',
      detail: `Latency OK (${avgP99.toFixed(0)}ms) but correctness ${avgCorr.toFixed(1)}% — unprotected concurrent order book writes.`,
    });
  }

  // RULE 5: Rejection flood
  if (totalFailed > totalTimeouts * 5 && totalFailed > 100) {
    cards.push({
      id: 'high-rejection', severity: 'warning',
      title: 'High Rejection Rate',
      detail: `${totalFailed.toLocaleString()} explicit 5xx/4xx rejections. Check for unhandled panics or invalid payload crashes.`,
    });
  }

  // RULE 6: Cold start penalty
  if (first.p50LatencyMs > last.p50LatencyMs * 2 && first.p50LatencyMs > 200) {
    cards.push({
      id: 'cold-start', severity: 'warning',
      title: 'Cold Start Penalty',
      detail: `Initial P50 ${first.p50LatencyMs.toFixed(0)}ms settled to ${last.p50LatencyMs.toFixed(0)}ms — JIT warmup or lazy init hurting score.`,
    });
  }

  // RULE 7: Throughput collapse
  if (tpsDrop > 0.35 && tpsStart > 10) {
    cards.push({
      id: 'tps-collapse', severity: tpsDrop > 0.6 ? 'critical' : 'warning',
      title: 'Throughput Collapse',
      detail: `TPS ${tpsStart.toFixed(0)} → ${tpsEnd.toFixed(0)} (${(tpsDrop*100).toFixed(0)}% drop) — memory leak, goroutine leak, or unbounded order book.`,
    });
  }

  // HEALTHY
  if (cards.length === 0) {
    cards.push({
      id: 'ok', severity: 'ok',
      title: 'No Issues Detected',
      detail: `Error ${overallErrPct.toFixed(2)}%  ·  P99 avg ${avgP99.toFixed(0)}ms  ·  Correctness ${avgCorr.toFixed(1)}%`,
    });
  }

  return cards;
}

// ── Severity badge ────────────────────────────────────────────────────────────

function SeverityChip({ s }: { s: Severity }) {
  const color = s === 'critical' ? 'var(--red)' : s === 'warning' ? 'var(--amber)' : 'var(--green)';
  const label = s === 'critical' ? 'CRIT' : s === 'warning' ? 'WARN' : 'OK';
  return <span className="diag-chip" style={{ background: color }}>{label}</span>;
}

// ── Main component ────────────────────────────────────────────────────────────

interface CachedDiagnosis {
  id: string;
  timestamp: string;
  snapshotCount: number;
  cards: DiagnosisCard[];
}

export function DiagnosisPanel({ snapshots, submissionId }: DiagnosisPanelProps) {
  const [history, setHistory] = useState<CachedDiagnosis[]>(() => {
    try {
      const stored = localStorage.getItem('bench_rca_history');
      return stored ? JSON.parse(stored) : [];
    } catch {
      return [];
    }
  });

  // Track last snapshot count so we only re-run when we actually have new data
  const lastCountRef = useRef(0);

  useEffect(() => {
    if (!submissionId || snapshots.length < 3) return;
    if (snapshots.length === lastCountRef.current) return;
    lastCountRef.current = snapshots.length;

    const newCards = diagnose(snapshots);
    if (newCards.length === 0) return;

    setHistory(prev => {
      const idx = prev.findIndex(h => h.id === submissionId);

      if (idx !== -1) {
        // Merge: keep existing cards and add any new card ids not yet seen
        const existingIds = new Set(prev[idx].cards.map(c => c.id));
        const merged = [
          ...prev[idx].cards,
          ...newCards.filter(c => !existingIds.has(c.id)),
        ];
        // Remove 'ok' if we now have real issues
        const hasIssues = merged.some(c => c.severity !== 'ok');
        const final = hasIssues ? merged.filter(c => c.id !== 'ok') : merged;
        const updated = prev.map((h, i) => i === idx
          ? { ...h, snapshotCount: snapshots.length, cards: final,
              timestamp: new Date().toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit' }) }
          : h
        );
        localStorage.setItem('bench_rca_history', JSON.stringify(updated));
        return updated;
      }

      // New submission entry
      const entry: CachedDiagnosis = {
        id: submissionId,
        timestamp: new Date().toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit' }),
        snapshotCount: snapshots.length,
        cards: newCards,
      };
      const updated = [entry, ...prev];
      localStorage.setItem('bench_rca_history', JSON.stringify(updated));
      return updated;
    });
  }, [snapshots, submissionId]);

  if (!submissionId) return null;
  if (history.length === 0) return null;

  return (
    <section className="diag-panel" id="diagnosis-panel">
      <h3 className="section-label">Root Cause Analysis</h3>
      <div className="diag-scroll">
        {history.map((h, hi) => (
          <div key={h.id} className="diag-run">
            <div className="diag-run-header">
              <span className="mono diag-run-id">#{history.length - hi} · {h.id.slice(0, 8)}</span>
              <span className="diag-run-meta">{h.snapshotCount} windows · {h.timestamp}</span>
            </div>
            <div className="diag-cards">
              {h.cards.map(c => (
                <div key={c.id} className={`diag-card diag-card--${c.severity}`}>
                  <SeverityChip s={c.severity} />
                  <span className="diag-card-title">{c.title}</span>
                  <span className="diag-card-detail">{c.detail}</span>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
