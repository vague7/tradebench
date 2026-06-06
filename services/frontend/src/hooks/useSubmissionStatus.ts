import { useEffect, useState } from 'react';

import { getSubmissionStatus } from '../api/client';
import type { Submission } from '../types/api';

export interface SubmissionStatusState {
  submission: Submission | null;
  loading: boolean;
  error: string | null;
}

export function useSubmissionStatus(submissionId: string | null, token: string, pollMs = 2000): SubmissionStatusState {
  const [submission, setSubmission] = useState<Submission | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!submissionId || !token) {
      setSubmission(null);
      return;
    }

    let cancelled = false;
    const load = async () => {
      setLoading(true);
      try {
        const next = await getSubmissionStatus(submissionId, token);
        if (!cancelled) {
          setSubmission(next);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Unable to load submission status');
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void load();
    const timer = window.setInterval(() => {
      void load();
    }, pollMs);

    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [submissionId, token, pollMs]);

  return { submission, loading, error };
}
