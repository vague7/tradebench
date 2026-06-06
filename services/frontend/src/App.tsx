import { useState } from 'react';

import { LeaderboardPage } from './pages/Leaderboard';
import { SubmitPage } from './pages/Submit';

type View = 'submit' | 'leaderboard';

export function App() {
  const [view, setView] = useState<View>('submit');

  return (
    <div className="app-shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Bench Platform</p>
          <h2>Distributed Benchmarking & Hosting</h2>
        </div>
        <nav className="switcher" aria-label="Primary">
          <button className={view === 'submit' ? 'active' : ''} onClick={() => setView('submit')} type="button">
            Submit
          </button>
          <button className={view === 'leaderboard' ? 'active' : ''} onClick={() => setView('leaderboard')} type="button">
            Leaderboard
          </button>
        </nav>
      </header>

      <main>{view === 'submit' ? <SubmitPage /> : <LeaderboardPage />}</main>
    </div>
  );
}
