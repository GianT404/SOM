import { Home, Library, ListMusic, Search, Settings } from 'lucide-react';
import type { ReactNode } from 'react';
import type { ViewKey } from '../lib/types';
import { usePlayer } from '../stores/playerStore';
import { LyricsPanel } from './LyricsPanel';
import { PlayerBar } from './PlayerBar';

interface AppShellProps {
  children: ReactNode;
  activeView: ViewKey;
  backendReady: boolean;
  backendUrl: string;
  onNavigate: (view: ViewKey) => void;
}

const navItems: Array<{ key: ViewKey; label: string; icon: typeof Home }> = [
  { key: 'home', label: 'Home', icon: Home },
  { key: 'search', label: 'Search', icon: Search },
  { key: 'library', label: 'Library', icon: Library },
  { key: 'lyrics', label: 'Lyrics', icon: ListMusic },
  { key: 'settings', label: 'Settings', icon: Settings },
];

export function AppShell({ children, activeView, backendReady, backendUrl, onNavigate }: AppShellProps) {
  const player = usePlayer();

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <button className="brand" onClick={() => onNavigate('home')} aria-label="Go home">
          <img src="/logo.png" alt="" />
          <span>SOM</span>
        </button>
        <nav className="nav-list" aria-label="Main">
          {navItems.map((item) => {
            const Icon = item.icon;
            return (
              <button
                key={item.key}
                className={`nav-item ${activeView === item.key ? 'active' : ''}`}
                onClick={() => onNavigate(item.key)}
              >
                <Icon size={20} />
                <span>{item.label}</span>
              </button>
            );
          })}
        </nav>
        <div className={`backend-chip ${backendReady ? 'ready' : 'offline'}`} title={backendUrl}>
          <span />
          {backendReady ? 'Local backend' : 'Backend offline'}
        </div>
      </aside>
      <main className="content">{children}</main>
      {activeView !== 'lyrics' && (
        <aside className="right-panel">
          <LyricsPanel compact />
        </aside>
      )}
      {player.currentTrack && <PlayerBar />}
    </div>
  );
}

