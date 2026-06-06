import type { SubmissionStatus } from '../types/api';

const STATUS_CONFIG: Record<SubmissionStatus, { label: string; className: string }> = {
  UPLOADED: { label: 'Uploaded', className: 'status-uploaded' },
  BUILDING: { label: 'Building', className: 'status-building' },
  RUNNING: { label: 'Running', className: 'status-running' },
  BENCHMARKING: { label: 'Benchmarking', className: 'status-benchmarking' },
  SCORED: { label: 'Scored', className: 'status-scored' },
  FAILED: { label: 'Failed', className: 'status-failed' },
};

export function StatusBadge({ status }: { status: SubmissionStatus }) {
  const config = STATUS_CONFIG[status];

  return (
    <span className={`status-badge ${config.className}`} aria-label={`Status: ${config.label}`}>
      <span className="status-dot" />
      {config.label}
    </span>
  );
}
