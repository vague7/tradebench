import { useCallback, useEffect, useRef, useState } from 'react';

import type { ApiHealthState } from '../types/api';

const API_BASE = import.meta.env.VITE_API_BASE ?? '';

/**
 * Polls the API gateway health endpoint every `intervalMs` and exposes
 * a simple online / offline / checking state for the header badge.
 */
export function useApiHealth(intervalMs = 15_000) {
  const [state, setState] = useState<ApiHealthState>('checking');
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const check = useCallback(async () => {
    try {
      // Try /api/health first; fall back to /api/leaderboard as a reachability probe.
      // Any HTTP response (even 404) proves the server is reachable → "online".
      // Only network-level failures (timeout, DNS, CORS) → "offline".
      const res = await fetch(`${API_BASE}/api/health`, {
        method: 'GET',
        signal: AbortSignal.timeout(5_000),
      });
      // 2xx = healthy, 404 = endpoint missing but server alive
      setState(res.ok || res.status === 404 ? 'online' : 'offline');
    } catch {
      setState('offline');
    }
  }, []);

  useEffect(() => {
    void check();
    timerRef.current = setInterval(() => void check(), intervalMs);
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [check, intervalMs]);

  return state;
}
