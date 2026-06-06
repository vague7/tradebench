import { useCallback, useEffect, useRef, useState } from 'react';

import { getLeaderboard } from '../api/client';
import { LeaderboardSSEClient } from '../api/sse';
import type { SSEConnectionState } from '../api/sse';
import { LeaderboardTable } from '../components/LeaderboardTable';
import type { LeaderboardEntry } from '../types/api';

export function LeaderboardPage() {
  const [entries, setEntries] = useState<LeaderboardEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [sseState, setSseState] = useState<SSEConnectionState>('disconnected');
  const [error, setError] = useState<string | null>(null);

  const clientRef = useRef<LeaderboardSSEClient | null>(null);

  // Initial REST fetch.
  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      setLoading(true);
      try {
        const data = await getLeaderboard();
        if (!cancelled) {
          setEntries(data);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load leaderboard');
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    void load();
    return () => { cancelled = true; };
  }, []);

  // SSE subscription.
  const handleUpdate = useCallback((next: LeaderboardEntry[]) => {
    setEntries(next);
  }, []);

  const handleStateChange = useCallback((state: SSEConnectionState) => {
    setSseState(state);
  }, []);

  const handleSSEError = useCallback((msg: string) => {
    setError(msg);
  }, []);

  useEffect(() => {
    const client = new LeaderboardSSEClient({
      onUpdate: handleUpdate,
      onStateChange: handleStateChange,
      onError: handleSSEError,
    });
    clientRef.current = client;
    client.connect();

    return () => {
      client.disconnect();
      clientRef.current = null;
    };
  }, [handleUpdate, handleStateChange, handleSSEError]);

  const connectionLabel = (): string => {
    switch (sseState) {
      case 'connected': return '● Live stream connected';
      case 'connecting': return '○ Connecting…';
      case 'error': return '● Stream disconnected';
      case 'disconnected': return '○ Stream not connected';
    }
  };

  return (
    <div className="page-grid leaderboard-layout">
      <section className="hero-panel panel">
        <p className="eyebrow">Live rankings</p>
        <h1>Real-time leaderboard powered by server-sent events.</h1>
        <p className="lead">
          Rankings update automatically as benchmarks complete. The SSE stream
          pushes new scores every 2 seconds while benchmarks are running.
        </p>
        <p className={`connection-indicator ${sseState}`} id="sse-status">{connectionLabel()}</p>
        {error ? <p className="form-error" role="alert">{error}</p> : null}
      </section>
      <LeaderboardTable entries={entries} loading={loading} />
    </div>
  );
}
