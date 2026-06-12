import type { MetricSnapshot, Score } from '../types/api';

interface MetricsPanelProps {
  snapshot: MetricSnapshot | null;
  score: Score | null;
}

type Severity = 'default' | 'green' | 'amber' | 'red';

interface MetricDef {
  label: string;
  value: string;
  unit?: string;
  severity: Severity;
}

function fmt(val: number | undefined | null, dec: number): string {
  if (val === undefined || val === null) return '—';
  return val.toFixed(dec);
}

/**
 * PRD sends correctnessScore as 0.0-1.0, but UI displays 0-100%.
 * Handle both scales safely: if value <= 1.0, multiply by 100.
 */
function normalizeCorrectness(raw: number | undefined): number | undefined {
  if (raw === undefined) return undefined;
  return raw <= 1.0 ? raw * 100 : raw;
}

function latSev(ms: number | undefined): Severity {
  if (ms === undefined) return 'default';
  if (ms < 100) return 'green';
  if (ms < 500) return 'amber';
  return 'red';
}

function correctSev(pct: number | undefined): Severity {
  if (pct === undefined) return 'default';
  if (pct >= 90) return 'green';
  if (pct >= 30) return 'amber';
  return 'red';
}

function errorSev(rate: number | undefined): Severity {
  if (rate === undefined) return 'default';
  if (rate < 1) return 'green';
  if (rate <= 5) return 'amber';
  return 'red';
}

export function MetricsPanel({ snapshot, score }: MetricsPanelProps) {
  const totalReqs = snapshot
    ? snapshot.successCount + snapshot.failureCount + snapshot.timeoutCount
    : 0;

  const errorRate = snapshot && totalReqs > 0
    ? ((snapshot.failureCount + snapshot.timeoutCount) / totalReqs) * 100
    : undefined;

  const normCorrectness = normalizeCorrectness(snapshot?.correctnessScore);
  const isDisqualified = score?.isDisqualified || (normCorrectness !== undefined && normCorrectness < 30);

  const metrics: MetricDef[] = [
    {
      label: 'Final Score',
      value: score ? (score.finalScore * 100).toFixed(1) : '—',
      unit: score ? 'pts' : undefined,
      severity: 'default',
    },
    {
      label: 'TPS',
      value: fmt(snapshot?.tps, 1),
      unit: snapshot ? 'req/s' : undefined,
      severity: snapshot && snapshot.tps >= 100 ? 'green' : snapshot ? 'amber' : 'default',
    },
    {
      label: 'P50 Latency',
      value: fmt(snapshot?.p50LatencyMs, 2),
      unit: 'ms',
      severity: latSev(snapshot?.p50LatencyMs),
    },
    {
      label: 'P90 Latency',
      value: fmt(snapshot?.p90LatencyMs, 2),
      unit: 'ms',
      severity: latSev(snapshot?.p90LatencyMs),
    },
    {
      label: 'P99 Latency',
      value: fmt(snapshot?.p99LatencyMs, 2),
      unit: 'ms',
      severity: latSev(snapshot?.p99LatencyMs),
    },
    {
      label: 'Error Rate',
      value: errorRate !== undefined ? errorRate.toFixed(2) : '—',
      unit: '%',
      severity: errorSev(errorRate),
    },
    {
      label: 'Correctness',
      value: fmt(normCorrectness, 1),
      unit: '%',
      severity: correctSev(normCorrectness),
    },
    {
      label: 'Success',
      value: snapshot ? String(snapshot.successCount) : '—',
      severity: 'default',
    },
    {
      label: 'Failure',
      value: snapshot ? String(snapshot.failureCount) : '—',
      severity: snapshot && snapshot.failureCount > 0 ? 'red' : 'default',
    },
    {
      label: 'Timeout',
      value: snapshot ? String(snapshot.timeoutCount) : '—',
      severity: snapshot && snapshot.timeoutCount > 0 ? 'amber' : 'default',
    },
  ];

  return (
    <section className="panel metrics-panel" id="metrics-panel">
      <div className="metrics-header">
        <h3 className="section-label">Benchmark Metrics</h3>
        {isDisqualified && <span className="dq-badge">DISQUALIFIED</span>}
      </div>

      <div className="metrics-grid">
        {metrics.map((m) => (
          <div className={`metric-card${m.severity !== 'default' ? ` metric--${m.severity}` : ''}`} key={m.label}>
            <span className="metric-label">{m.label}</span>
            <span className="metric-value">
              {m.value}
              {m.unit && m.value !== '—' && <span className="metric-unit"> {m.unit}</span>}
            </span>
          </div>
        ))}
      </div>

      {score?.isDisqualified && (
        <div className="dq-reason" role="alert">
          <strong>Reason:</strong> {score.disqualifyReason ?? 'Correctness below 30% threshold'}
        </div>
      )}
    </section>
  );
}
