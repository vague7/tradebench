import type { MetricSnapshot, Score } from '../types/api';

interface MetricsPanelProps {
  snapshot: MetricSnapshot | null;
  score: Score | null;
}

interface MetricCardData {
  label: string;
  value: string;
  unit?: string;
}

function formatMetric(val: number | undefined | null, decimals: number, fallback = '—'): string {
  if (val === undefined || val === null) return fallback;
  return val.toFixed(decimals);
}

export function MetricsPanel({ snapshot, score }: MetricsPanelProps) {
  const cards: MetricCardData[] = [
    { label: 'P50 Latency', value: formatMetric(snapshot?.p50LatencyMs, 2), unit: 'ms' },
    { label: 'P90 Latency', value: formatMetric(snapshot?.p90LatencyMs, 2), unit: 'ms' },
    { label: 'P99 Latency', value: formatMetric(snapshot?.p99LatencyMs, 2), unit: 'ms' },
    { label: 'TPS', value: formatMetric(snapshot?.tps, 2) },
    { label: 'Success Count', value: snapshot ? String(snapshot.successCount) : '—' },
    { label: 'Failure Count', value: snapshot ? String(snapshot.failureCount) : '—' },
    { label: 'Timeout Count', value: snapshot ? String(snapshot.timeoutCount) : '—' },
    { label: 'Correctness', value: formatMetric(snapshot?.correctnessScore, 2), unit: '%' },
  ];

  return (
    <section className="panel metrics-section" id="metrics-panel">
      <h3 className="form-title">Performance metrics</h3>
      <div className="metrics-grid">
        {cards.map((card) => (
          <article className="metric-card" key={card.label}>
            <span className="metric-label">{card.label}</span>
            <strong className="metric-value">
              {card.value}
              {card.unit && card.value !== '—' ? <span className="metric-unit"> {card.unit}</span> : null}
            </strong>
          </article>
        ))}
      </div>
      {score ? (
        <div className="score-summary">
          <div className="score-item">
            <span className="metric-label">Final Score</span>
            <strong className="score-value">{(score.finalScore * 100).toFixed(1)}</strong>
          </div>
          {score.isDisqualified ? (
            <p className="form-error" role="alert">
              Disqualified: {score.disqualifyReason ?? 'Unknown reason'}
            </p>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}
