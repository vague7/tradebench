import type { ConnectionState, ApiHealthState } from '../types/api';

interface ConnectionBadgeProps {
  label: string;
  state: ConnectionState;
  lastUpdated?: Date | null;
}

export function ConnectionBadge({ label, state, lastUpdated }: ConnectionBadgeProps) {
  const stateLabel: Record<ConnectionState, string> = {
    connected: 'Live',
    connecting: 'Connecting',
    reconnecting: 'Reconnecting',
    offline: 'Offline',
  };

  const elapsed = lastUpdated ? formatElapsed(Date.now() - lastUpdated.getTime()) : null;

  return (
    <div className={`conn-badge conn-badge--${state}`} title={`${label}: ${stateLabel[state]}`}>
      <span className="conn-dot" />
      <span className="conn-text">{label}</span>
      {elapsed && state === 'connected' && (
        <span className="conn-time">{elapsed}</span>
      )}
    </div>
  );
}

interface ApiHealthBadgeProps {
  state: ApiHealthState;
}

export function ApiHealthBadge({ state }: ApiHealthBadgeProps) {
  const labels: Record<ApiHealthState, string> = {
    online: 'API Online',
    offline: 'API Offline',
    checking: 'Checking…',
  };

  const connectionState: Record<ApiHealthState, ConnectionState> = {
    online: 'connected',
    offline: 'offline',
    checking: 'connecting',
  };

  return (
    <div className={`conn-badge conn-badge--${connectionState[state]}`} title={labels[state]}>
      <span className="conn-dot" />
      <span className="conn-text">{labels[state]}</span>
    </div>
  );
}

function formatElapsed(ms: number): string {
  const s = Math.floor(ms / 1000);
  if (s < 5) return 'just now';
  if (s < 60) return `${s}s ago`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  return `${Math.floor(m / 60)}h ago`;
}
