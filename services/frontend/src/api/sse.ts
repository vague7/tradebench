import type { LeaderboardEntry, LeaderboardStreamPayload } from '../types/api';

const API_BASE = import.meta.env.VITE_API_BASE ?? '';

export type SSEConnectionState = 'disconnected' | 'connecting' | 'connected' | 'error';

export interface SSEEventHandlers {
  onUpdate: (entries: LeaderboardEntry[]) => void;
  onStateChange?: (state: SSEConnectionState) => void;
  onError?: (message: string) => void;
}

/**
 * Reusable SSE transport wrapper.
 *
 * Handles connection lifecycle (connect / disconnect / reconnect) for the
 * leaderboard event stream. Contains zero business logic — only transport
 * concerns. UI layers subscribe via the callback interface.
 */
export class LeaderboardSSEClient {
  private source: EventSource | null = null;
  private handlers: SSEEventHandlers;
  private url: string;
  private state: SSEConnectionState = 'disconnected';

  constructor(handlers: SSEEventHandlers, url?: string) {
    this.handlers = handlers;
    this.url = url ?? `${API_BASE}/api/leaderboard/stream`;
  }

  /** Open the SSE connection. Safe to call when already connected (no-op). */
  connect(): void {
    if (this.source) {
      return;
    }

    this.setState('connecting');

    const source = new EventSource(this.url);

    source.onopen = () => {
      this.setState('connected');
    };

    source.onerror = () => {
      this.setState('error');
      this.handlers.onError?.('Leaderboard stream disconnected');
    };

    source.addEventListener('leaderboard_update', (event) => {
      const message = event as MessageEvent<string>;
      try {
        const payload = JSON.parse(message.data) as LeaderboardStreamPayload;
        this.handlers.onUpdate(payload.rankings);
      } catch {
        this.handlers.onError?.('Failed to parse leaderboard update');
      }
    });

    this.source = source;
  }

  /** Close the SSE connection gracefully. */
  disconnect(): void {
    if (this.source) {
      this.source.close();
      this.source = null;
    }
    this.setState('disconnected');
  }

  /** Drop the current connection and open a fresh one. */
  reconnect(): void {
    this.disconnect();
    this.connect();
  }

  /** Get the current connection state. */
  getState(): SSEConnectionState {
    return this.state;
  }

  private setState(next: SSEConnectionState): void {
    this.state = next;
    this.handlers.onStateChange?.(next);
  }
}
