import type { SubmissionStatus } from '../types/api';
import { BenchmarkStageInfo } from './BenchmarkStageInfo';

const PIPELINE_STEPS: { status: SubmissionStatus; label: string; description: string }[] = [
  { status: 'UPLOADED', label: 'Upload', description: 'ZIP received and validated' },
  { status: 'BUILDING', label: 'Build', description: 'Docker image compiled' },
  { status: 'RUNNING', label: 'Sandbox', description: 'Container started on bench-net' },
  { status: 'BENCHMARKING', label: 'Benchmark', description: 'Load test running' },
  { status: 'SCORED', label: 'Score', description: 'Results computed & ranked' },
];

const STATUS_ORDER: Record<SubmissionStatus, number> = {
  UPLOADED: 0,
  BUILDING: 1,
  RUNNING: 2,
  BENCHMARKING: 3,
  SCORED: 4,
  FAILED: -1,
};

interface SubmissionPipelineProps {
  /** null = preview mode (no active submission) */
  currentStatus: SubmissionStatus | null;
}

export function SubmissionPipeline({ currentStatus }: SubmissionPipelineProps) {
  const isPreview = currentStatus === null;
  const currentIndex = currentStatus ? (currentStatus === 'SCORED' ? 5 : STATUS_ORDER[currentStatus]) : -1;
  const isFailed = currentStatus === 'FAILED';

  return (
    <div className="pipeline" id="pipeline-tracker">
      <div className="pipeline-steps">
        {PIPELINE_STEPS.map((step, i) => {
          let cls = 'pipeline-step';
          if (isPreview) {
            cls += ' pipeline-step--preview';
          } else if (isFailed) {
            cls += ' pipeline-step--failed';
          } else if (i < currentIndex) {
            cls += ' pipeline-step--done';
          } else if (i === currentIndex) {
            cls += ' pipeline-step--active';
          } else {
            cls += ' pipeline-step--pending';
          }

          return (
            <div key={step.status} className={cls}>
              {/* connector line */}
              {i > 0 && (
                <div className={`pipeline-line${!isPreview && !isFailed && i <= currentIndex ? ' pipeline-line--filled' : ''}${isFailed ? ' pipeline-line--failed' : ''}`} />
              )}
              {/* node */}
              <div className="pipeline-node">
                {!isPreview && !isFailed && i < currentIndex ? (
                  <svg width="12" height="12" viewBox="0 0 12 12" fill="none"><path d="M2 6L5 9L10 3" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/></svg>
                ) : isFailed ? (
                  <svg width="10" height="10" viewBox="0 0 10 10" fill="none"><path d="M2 2L8 8M8 2L2 8" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"/></svg>
                ) : !isPreview && i === currentIndex ? (
                  <span className="pipeline-pulse" />
                ) : (
                  <span className="pipeline-num">{i + 1}</span>
                )}
              </div>
              {/* label + desc */}
              <div className="pipeline-text">
                <span className="pipeline-label">{step.label}</span>
                <span className="pipeline-desc">{step.description}</span>
              </div>
            </div>
          );
        })}
      </div>
    <BenchmarkStageInfo currentStatus={currentStatus} />
  </div>
  );
}
