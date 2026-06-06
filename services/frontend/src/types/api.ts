export type SubmissionStatus =
  | 'UPLOADED'
  | 'BUILDING'
  | 'RUNNING'
  | 'BENCHMARKING'
  | 'SCORED'
  | 'FAILED';

export interface Submission {
  id: string;
  teamName: string;
  status: SubmissionStatus;
  errorMessage?: string;
  uploadedAt: string;
  benchmarkStart?: string;
  benchmarkEnd?: string;
}

export interface MetricSnapshot {
  submissionId: string;
  windowEnd: string;
  p50LatencyMs: number;
  p90LatencyMs: number;
  p99LatencyMs: number;
  tps: number;
  successCount: number;
  failureCount: number;
  timeoutCount: number;
  correctnessScore: number;
}

export interface Score {
  submissionId: string;
  teamName: string;
  throughputScore: number;
  latencyScore: number;
  correctnessScore: number;
  finalScore: number;
  isDisqualified: boolean;
  disqualifyReason?: string;
  computedAt: string;
}

export interface LeaderboardEntry {
  rank: number;
  teamName: string;
  finalScore: number;
  tps: number;
  p99LatencyMs: number;
  errorRate: number;
  correctnessScore: number;
  status: SubmissionStatus;
}

export interface ApiErrorShape {
  error: string;
  code: string;
}

export interface SubmissionUploadResponse {
  submissionId: string;
}

export interface SubmissionResults {
  snapshot: MetricSnapshot;
  score: Score;
}

export interface LeaderboardStreamPayload {
  event: 'leaderboard_update';
  timestamp: string;
  rankings: LeaderboardEntry[];
}
