import type { MetricSnapshot, Score } from '../types/api';

export function MetricsPanel({ snapshot, score }: { snapshot: MetricSnapshot | null; score: Score | null }) {
  const cards = [
    { label: 'P99 latency', value: snapshot ? `${snapshot.p99LatencyMs.toFixed(2)} ms` : '—' },
    { label: 'TPS', value: snapshot ? snapshot.tps.toFixed(2) : '—' },
    { label: 'Correctness', value: snapshot ? `${snapshot.correctnessScore.toFixed(2)}%` : '—' },
    { label: 'Final score', value: score ? score.finalScore.toFixed(2) : '—' },
  ];

  return (
    <section className="metrics-grid">
      {cards.map((card) => (
        <article className="metric-card" key={card.label}>
          <span className="metric-label">{card.label}</span>
          <strong className="metric-value">{card.value}</strong>
        </article>
      ))}
    </section>
  );
}
