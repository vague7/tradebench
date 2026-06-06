import { useMemo } from 'react';

import { LeaderboardTable } from '../components/LeaderboardTable';
import { useLeaderboardStream } from '../api/sse';

export function LeaderboardPage() {
  const { entries, connected, error } = useLeaderboardStream();
  const subtitle = useMemo(() => {
    if (connected) {
      return 'Live SSE stream connected';
    }
    return 'Waiting for leaderboard updates';
  }, [connected]);

  return (
    <div className="page-grid leaderboard-layout">
      <section className="hero-panel panel">
        <p className="eyebrow">Live rankings</p>
        <h1>Real-time leaderboard with streamed updates.</h1>
        <p className="lead">This view listens to the server-sent event stream and keeps the ranking table synchronized.</p>
        <p className="inline-note">{subtitle}</p>
        {error ? <p className="form-error">{error}</p> : null}
      </section>
      <LeaderboardTable entries={entries} />
    </div>
  );
}
