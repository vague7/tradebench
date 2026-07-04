import { useCallback, useEffect, useRef, useState } from 'react';

import { getLeaderboard } from '../api/client';
import { LeaderboardSSEClient } from '../api/sse';
import type { SSEConnectionState } from '../api/sse';
import type { ConnectionState, LeaderboardEntry } from '../types/api';

function toConnectionState(sse: SSEConnectionState): ConnectionState {
  switch (sse) {
    case 'connected': return 'connected';
    case 'connecting': return 'connecting';
    case 'error': return 'reconnecting';
    case 'disconnected': return 'offline';
  }
}

export interface LeaderboardStreamState {
  entries: LeaderboardEntry[];
  loading: boolean;
  connection: ConnectionState;
  lastUpdated: Date | null;
  error: string | null;
  retry: () => void;
}

/**
 * Encapsulates the full leaderboard data flow:
 *   1. Initial REST fetch on mount
 *   2. SSE subscription for live updates
 *   3. Connection state tracking with last-updated timestamp
 *   4. Manual retry action
 */
export function useLeaderboardStream(): LeaderboardStreamState {
  const [entries, setEntries] = useState<LeaderboardEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [connection, setConnection] = useState<ConnectionState>('connecting');
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [error, setError] = useState<string | null>(null);

  const clientRef = useRef<LeaderboardSSEClient | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getLeaderboard();
      setEntries(data || []);
      setLastUpdated(new Date());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load leaderboard');
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial REST fetch.
  useEffect(() => {
    void loadData();
  }, [loadData]);

  // SSE subscription.
  const handleUpdate = useCallback((next: LeaderboardEntry[]) => {
    setEntries(next || []);
    setLastUpdated(new Date());
    setError(null);
  }, []);

  const handleStateChange = useCallback((state: SSEConnectionState) => {
    setConnection(toConnectionState(state));
  }, []);

  const handleSSEError = useCallback((_msg: string) => {
    // Don't overwrite error if we already have one from initial load
    // SSE errors are reflected in connection state instead
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

  const retry = useCallback(() => {
    clientRef.current?.reconnect();
    void loadData();
  }, [loadData]);

  return { entries, loading, connection, lastUpdated, error, retry };
}
