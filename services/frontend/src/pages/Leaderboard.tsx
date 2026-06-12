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
      {!loading && entries.length === 0 && !error ? (
        <EmptyState
          icon={
            <svg width="36" height="36" viewBox="0 0 36 36" fill="none">
              <rect x="4" y="6" width="28" height="24" rx="3" stroke="currentColor" strokeWidth="1.8"/>
              <path d="M4 12h28" stroke="currentColor" strokeWidth="1.8"/>
              <path d="M12 18h12M12 23h8" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"/>
            </svg>
          }
          title="No scored submissions yet."
          description="Upload a valid exchange to start the benchmark pipeline."
          action={onNavigateToSubmit ? { label: 'Go to Submit', onClick: onNavigateToSubmit } : undefined}
        />
      ) : (
        <LeaderboardTable entries={entries} loading={loading} />
      )}
    </div>
  );
}
