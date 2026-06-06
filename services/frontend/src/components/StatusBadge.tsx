import type { SubmissionStatus } from '../types/api';

const LABELS: Record<SubmissionStatus, string> = {
  UPLOADED: 'Uploaded',
  BUILDING: 'Building',
  RUNNING: 'Running',
  BENCHMARKING: 'Benchmarking',
  SCORED: 'Scored',
  FAILED: 'Failed',
};

export function StatusBadge({ status }: { status: SubmissionStatus }) {
  return <span className={`status-badge status-${status.toLowerCase()}`}>{LABELS[status]}</span>;
}
