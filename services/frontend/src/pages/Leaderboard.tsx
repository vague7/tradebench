import { ConnectionBadge } from '../components/ConnectionBadge';
import { EmptyState } from '../components/EmptyState';
import { ErrorBanner } from '../components/ErrorBanner';
import { LeaderboardTable } from '../components/LeaderboardTable';
import { useLeaderboardStream } from '../hooks/useLeaderboardStream';

interface LeaderboardPageProps {
  onNavigateToSubmit?: () => void;
}

export function LeaderboardPage({ onNavigateToSubmit }: LeaderboardPageProps) {
  const { entries, loading, connection, lastUpdated, error, retry } = useLeaderboardStream();

  return (
    <div className="leaderboard-page">
      {/* Compact header */}
      <div className="lb-header">
        <div className="lb-header-left">
          <h2 className="section-label" style={{ margin: 0, fontSize: '1rem' }}>Leaderboard</h2>
          <span className="lb-count">{entries.length} teams</span>
        </div>
        <ConnectionBadge label="SSE" state={connection} lastUpdated={lastUpdated} />
      </div>

      {/* Error banner — only when offline, not when just connecting */}
      {error && connection !== 'connecting' && !loading && (
        <ErrorBanner
          title="Leaderboard unavailable"
          message="Retrying connection to the API…"
          detail={error}
          lastSuccess={lastUpdated}
          onRetry={retry}
          onDismiss={undefined}
        />
      )}

      {/* Content */}
      <LeaderboardTable entries={entries} loading={loading} />
    </div>
  );
}
