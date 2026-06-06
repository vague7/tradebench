import { useCallback, useEffect, useRef, useState } from 'react';

import { getSubmissionStatus } from '../api/client';
import type { Submission, SubmissionStatus } from '../types/api';

/** Phases the polling lifecycle can be in. */
export type PollingPhase =
  | 'idle'
  | 'loading'
  | 'polling'
  | 'success'
  | 'failed'
  | 'timeout';

export interface SubmissionStatusState {
  /** Latest submission snapshot returned by the API. */
  submission: Submission | null;
  /** Current lifecycle phase. */
  phase: PollingPhase;
  /** Human-readable error message, if any. */
  error: string | null;
}

interface PollingOptions {
  /** Milliseconds between polls. @default 2000 */
  intervalMs?: number;
  /** Maximum milliseconds before the hook declares a timeout. @default 300_000 (5 min) */
  timeoutMs?: number;
}

/** Terminal statuses that should stop the polling loop. */
const TERMINAL_STATUSES: ReadonlySet<SubmissionStatus> = new Set<SubmissionStatus>([
  'SCORED',
  'FAILED',
]);

/**
 * Polls `GET /api/submissions/:id/status` and exposes a phase-aware state
 * object for UI consumption.
 *
 * The hook transitions through a well-defined lifecycle:
 *   idle → loading → polling → success | failed | timeout
 *
 * Polling stops automatically when the submission reaches a terminal status
 * (SCORED or FAILED) or when the configurable timeout elapses.
 */
export function useSubmissionStatus(
  submissionId: string | null,
  token: string,
  options?: PollingOptions,
): SubmissionStatusState {
  const intervalMs = options?.intervalMs ?? 2_000;
  const timeoutMs = options?.timeoutMs ?? 300_000;

  const [submission, setSubmission] = useState<Submission | null>(null);
  const [phase, setPhase] = useState<PollingPhase>('idle');
  const [error, setError] = useState<string | null>(null);

  // Track the start time for timeout calculation.
  const startRef = useRef<number>(0);

  const poll = useCallback(
    async (id: string, tk: string): Promise<boolean> => {
      try {
        const next = await getSubmissionStatus(id, tk);
        setSubmission(next);
        setError(null);

        if (TERMINAL_STATUSES.has(next.status)) {
          setPhase(next.status === 'SCORED' ? 'success' : 'failed');
          return false; // stop polling
        }

        // Check for timeout.
        if (Date.now() - startRef.current >= timeoutMs) {
          setPhase('timeout');
          setError('Status polling timed out');
          return false; // stop polling
        }

        setPhase('polling');
        return true; // continue polling
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unable to load submission status');
        setPhase('failed');
        return false; // stop polling on error
      }
    },
    [timeoutMs],
  );

  useEffect(() => {
    if (!submissionId || !token) {
      setSubmission(null);
      setPhase('idle');
      setError(null);
      return;
    }

    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | null = null;

    startRef.current = Date.now();
    setPhase('loading');

    const tick = async () => {
      if (cancelled) return;
      const shouldContinue = await poll(submissionId, token);
      if (!cancelled && shouldContinue) {
        timer = setTimeout(() => void tick(), intervalMs);
      }
    };

    void tick();

    return () => {
      cancelled = true;
      if (timer !== null) {
        clearTimeout(timer);
      }
    };
  }, [submissionId, token, intervalMs, poll]);

  return { submission, phase, error };
}
