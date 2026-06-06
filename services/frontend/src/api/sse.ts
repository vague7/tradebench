import { useEffect, useState } from 'react';

import type { LeaderboardEntry, LeaderboardStreamPayload } from '../types/api';

const API_BASE = import.meta.env.VITE_API_BASE ?? '';

export interface LeaderboardStreamState {
  entries: LeaderboardEntry[];
  connected: boolean;
  error: string | null;
}

export function useLeaderboardStream(): LeaderboardStreamState {
  const [entries, setEntries] = useState<LeaderboardEntry[]>([]);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const source = new EventSource(`${API_BASE}/api/leaderboard/stream`);

    source.onopen = () => {
      setConnected(true);
      setError(null);
    };

    source.onerror = () => {
      setConnected(false);
      setError('Leaderboard stream disconnected');
    };

    source.addEventListener('leaderboard_update', (event) => {
      const message = event as MessageEvent<string>;
      try {
        const payload = JSON.parse(message.data) as LeaderboardStreamPayload;
        setEntries(payload.rankings);
      } catch {
        setError('Failed to parse leaderboard update');
      }
    });

    return () => {
      source.close();
    };
  }, []);

  return { entries, connected, error };
}
