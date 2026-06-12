import type { BenchmarkPhase } from '../types/api';

const PHASES: { id: BenchmarkPhase; label: string; desc: string }[] = [
  { id: 'warm-up', label: 'Warm-up', desc: '~10s low traffic' },
  { id: 'ramp', label: 'Ramp', desc: 'Linear increase' },
  { id: 'sustained', label: 'Sustained', desc: 'Peak steady state' },
  { id: 'spike', label: 'Spike', desc: '2× burst load' },
  { id: 'drain', label: 'Drain', desc: 'Graceful wind-down' },
];

interface BenchmarkPhaseTrackerProps {
  /** Currently active phase, or null for preview mode */
  activePhase?: BenchmarkPhase | null;
}

export function BenchmarkPhaseTracker({ activePhase = null }: BenchmarkPhaseTrackerProps) {
  return (
    <div className="phase-tracker">
      <h4 className="section-label">Benchmark Phases</h4>
      <div className="phase-steps">
        {PHASES.map((p) => {
          let cls = 'phase-step';
          if (activePhase === null) {
            cls += ' phase-step--preview';
          } else if (p.id === activePhase) {
            cls += ' phase-step--active';
          } else {
            cls += ' phase-step--muted';
          }

          return (
            <div key={p.id} className={cls}>
              <span className="phase-dot" />
              <span className="phase-name">{p.label}</span>
              <span className="phase-desc">{p.desc}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
