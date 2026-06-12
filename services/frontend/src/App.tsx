import { useState } from 'react';

import { ApiHealthBadge } from './components/ConnectionBadge';
import { ToastContainer } from './components/Toast';
import { useApiHealth } from './hooks/useApiHealth';
import { LeaderboardPage } from './pages/Leaderboard';
import { SubmitPage } from './pages/Submit';

type View = 'submit' | 'leaderboard';

export function App() {
  const [view, setView] = useState<View>('submit');
  const apiHealth = useApiHealth();

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="topbar-left">
          <div className="topbar-brand">
            <svg width="22" height="22" viewBox="0 0 100 100" fill="none" xmlns="http://www.w3.org/2000/svg">
              <defs>
                <linearGradient id="og" x1="0" y1="0" x2="100" y2="100" gradientUnits="userSpaceOnUse">
                  <stop stopColor="#FF7A00"/>
                  <stop offset="1" stopColor="#FF3D00"/>
                </linearGradient>
              </defs>
              <path d="M45 20 H65 C85 20 85 45 65 45 H45 Z" fill="url(#og)"/>
              <path d="M40 45 H70 C95 45 95 80 70 80 H40 Z" fill="url(#og)"/>
              <rect x="10" y="30" width="30" height="10" rx="5" fill="url(#og)"/>
              <rect x="0" y="50" width="45" height="10" rx="5" fill="url(#og)"/>
              <rect x="15" y="70" width="25" height="10" rx="5" fill="url(#og)"/>
            </svg>
            <span className="topbar-title">Tradebench</span>
          </div>

          <nav className="switcher" aria-label="Primary navigation">
            <button
              id="nav-submit"
              className={view === 'submit' ? 'active' : ''}
              onClick={() => setView('submit')}
              type="button"
            >
              Submit
            </button>
            <button
              id="nav-leaderboard"
              className={view === 'leaderboard' ? 'active' : ''}
              onClick={() => setView('leaderboard')}
              type="button"
            >
              Leaderboard
            </button>
          </nav>
        </div>

        <div className="topbar-right">
          <ApiHealthBadge state={apiHealth} />
        </div>
      </header>

      <main>
        {view === 'submit'
          ? <SubmitPage />
          : <LeaderboardPage onNavigateToSubmit={() => setView('submit')} />
        }
      </main>

      <ToastContainer />
    </div>
  );
}
