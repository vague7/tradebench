import type { SubmissionStatus } from '../types/api';

const STATUS_CONFIG: Record<SubmissionStatus, { label: string; className: string }> = {
  UPLOADED: { label: 'Uploaded', className: 'pill--blue' },
  BUILDING: { label: 'Building', className: 'pill--amber pill--pulse' },
  RUNNING: { label: 'Running', className: 'pill--green pill--pulse' },
  BENCHMARKING: { label: 'Benchmarking', className: 'pill--purple pill--pulse' },
  SCORED: { label: 'Scored', className: 'pill--green' },
  FAILED: { label: 'Failed', className: 'pill--red' },
};

export function StatusPill({ status }: { status: SubmissionStatus; }) {
  const config = STATUS_CONFIG[status];
  return (
    <span className={`status-pill ${config.className}`} aria-label={`Status: ${config.label}`}>
      <span className="pill-dot" />
      {config.label}
    </span>
  );
}

export function DisqualifiedPill() {
  return (
    <span className="status-pill pill--red" aria-label="Disqualified">
      <span className="pill-dot" />
      DQ
    </span>
  );
}
