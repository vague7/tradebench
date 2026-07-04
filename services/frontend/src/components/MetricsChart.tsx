import type { MetricSnapshot } from '../types/api';

interface MetricsChartProps {
  snapshots: MetricSnapshot[];
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function norm(v: number): number {
  return v <= 1.0 ? v * 100 : v;
}

function errorRate(s: MetricSnapshot): number {
  const total = s.successCount + s.failureCount + s.timeoutCount;
  return total > 0 ? ((s.failureCount + s.timeoutCount) / total) * 100 : 0;
}

function toCoords(
  values: number[],
  w: number,
  h: number,
  pad: number,
): { xs: number[]; ys: number[]; points: string } {
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const xs = values.map((_, i) => pad + (i / Math.max(values.length - 1, 1)) * (w - pad * 2));
  const ys = values.map((v) => h - pad - ((v - min) / range) * (h - pad * 2));
  return { xs, ys, points: xs.map((x, i) => `${x.toFixed(1)},${ys[i].toFixed(1)}`).join(' ') };
}

function yLabel(v: number, unit: string): string {
  // Never abbreviate ms — 1400ms is clearer than 1.4kms
  if (unit !== 'ms' && v >= 1000) return `${(v / 1000).toFixed(1)}k${unit}`;
  return `${v.toFixed(v < 10 ? 1 : 0)}${unit}`;
}

// Worst-point: index of the extreme value for each panel
function worstIdx(values: number[], mode: 'max' | 'min'): number {
  let idx = 0;
  values.forEach((v, i) => {
    if (mode === 'max' ? v > values[idx] : v < values[idx]) idx = i;
  });
  return idx;
}

// ── Main component ────────────────────────────────────────────────────────────

export function MetricsChart({ snapshots }: MetricsChartProps) {
  if (snapshots.length < 2) {
    return (
      <div className="chart-empty">
        <span>Waiting for benchmark data…</span>
      </div>
    );
  }

  const W = 520;
  const H = 110;
  const PAD = 28;

  const errRates  = snapshots.map(errorRate);
  const normCorr  = snapshots.map((s) => norm(s.correctnessScore));
  const timestamps = snapshots.map((s) =>
    new Date(s.windowEnd).toLocaleTimeString('en-US', {
      hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit',
    }),
  );

  const first = timestamps[0];
  const last  = timestamps[timestamps.length - 1];

  type Panel = {
    label: string;
    color: string;
    values: number[];
    unit: string;
    worstMode: 'max' | 'min';
    worstLabel: string;
  };

  const panels: Panel[] = [
    {
      label: 'TPS', color: 'var(--accent)',
      values: snapshots.map((s) => s.tps),
      unit: '', worstMode: 'min', worstLabel: 'TPS drop',
    },
    {
      label: 'P99 (ms)', color: 'var(--blue)',
      values: snapshots.map((s) => s.p99LatencyMs),
      unit: 'ms', worstMode: 'max', worstLabel: 'Peak P99',
    },
    {
      label: 'Error %', color: 'var(--red)',
      values: errRates,
      unit: '%', worstMode: 'max', worstLabel: 'Peak errors',
    },
    {
      label: 'Correct %', color: 'var(--green)',
      values: normCorr,
      unit: '%', worstMode: 'min', worstLabel: 'Correctness drop',
    },
  ];

  return (
    <section className="panel chart-panel" id="metrics-chart">
      <div className="chart-header">
        <h3 className="section-label">Live Metrics Timeline</h3>
        <span className="chart-range mono">{first} → {last} · {snapshots.length} windows</span>
      </div>

      <div className="chart-grid">
        {panels.map((p) => {
          const min   = Math.min(...p.values);
          const max   = Math.max(...p.values);
          const latest = p.values[p.values.length - 1];
          const { xs, ys, points } = toCoords(p.values, W, H, PAD);

          // Only show worst-point annotation if there is meaningful variance
          const range = max - min;
          const wi = worstIdx(p.values, p.worstMode);
          const showAnnotation = range > 0 && wi !== p.values.length - 1;

          return (
            <div key={p.label} className="chart-cell">
              <div className="chart-cell-header">
                <span className="chart-cell-label" style={{ color: p.color }}>{p.label}</span>
                <span className="chart-cell-latest mono">{yLabel(latest, p.unit)}</span>
              </div>

              <svg
                viewBox={`0 0 ${W} ${H}`}
                width="100%"
                height={H}
                preserveAspectRatio="none"
                className="chart-svg"
              >
                {/* Grid lines */}
                <line x1={PAD} y1={PAD}     x2={W - PAD} y2={PAD}     stroke="var(--border)" strokeWidth="1" />
                <line x1={PAD} y1={H - PAD} x2={W - PAD} y2={H - PAD} stroke="var(--border)" strokeWidth="1" />

                {/* Fill */}
                {points && (
                  <polyline
                    points={`${PAD},${H - PAD} ${points} ${W - PAD},${H - PAD}`}
                    fill={p.color} fillOpacity="0.08" stroke="none"
                  />
                )}

                {/* Line */}
                {points && (
                  <polyline
                    points={points} fill="none"
                    stroke={p.color} strokeWidth="1.8"
                    strokeLinejoin="round" strokeLinecap="round"
                  />
                )}

                {/* Latest-point dot */}
                <circle cx={xs[xs.length - 1]} cy={ys[ys.length - 1]} r="3" fill={p.color} />

                {/* Worst-point annotation */}
                {showAnnotation && (() => {
                  const ax = xs[wi];
                  const ay = ys[wi];
                  const above = ay > H / 2; // label goes above or below
                  const ly = above ? ay - 10 : ay + 18;
                  return (
                    <g>
                      <circle cx={ax} cy={ay} r="4" fill="none" stroke={p.color} strokeWidth="1.5" />
                      <circle cx={ax} cy={ay} r="1.5" fill={p.color} />
                      <text
                        x={Math.min(Math.max(ax, PAD + 10), W - PAD - 10)}
                        y={ly}
                        fontSize="8"
                        fill={p.color}
                        textAnchor="middle"
                        opacity="0.9"
                      >
                        {p.worstLabel}
                      </text>
                    </g>
                  );
                })()}

                {/* Y-axis labels */}
                <text x={PAD - 4} y={PAD + 4}     fontSize="9" fill="var(--text-3)" textAnchor="end">{yLabel(max, p.unit)}</text>
                <text x={PAD - 4} y={H - PAD + 4} fontSize="9" fill="var(--text-3)" textAnchor="end">{yLabel(min, p.unit)}</text>

                {/* Tooltip via SVG title on each data point */}
                {xs.map((x, i) => (
                  <circle key={i} cx={x} cy={ys[i]} r="5" fill="transparent" style={{ cursor: 'crosshair' }}>
                    <title>{timestamps[i]}: {yLabel(p.values[i], p.unit)}</title>
                  </circle>
                ))}
              </svg>
            </div>
          );
        })}
      </div>

      <div className="chart-xaxis">
        <span className="mono">{first}</span>
        <span className="mono">{last}</span>
      </div>
    </section>
  );
}
