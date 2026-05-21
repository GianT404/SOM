import { Bell, ChevronLeft, ChevronRight, Home, Library, ListMusic, Search, Settings, ShoppingCart, UserRound, UsersRound, X } from 'lucide-react';
import type { ReactNode } from 'react';
import { useEffect, useState } from 'react';
import { getPlaylist } from '../lib/storage';
import type { OfflineTrack, ViewKey } from '../lib/types';
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
  const [playlist, setPlaylist] = useState<OfflineTrack[]>([]);

  useEffect(() => {
    const load = () => setPlaylist(getPlaylist());
    load();
    window.addEventListener('som-playlist-changed', load);
    return () => window.removeEventListener('som-playlist-changed', load);
  }, []);

  return (
    <div className="app-shell">
      <aside className="icon-rail frame" aria-label="Main">
        <span className="frame-label">Rail</span>
        <nav className="rail-nav" aria-label="Main navigation">
          {navItems.map((item) => {
            const Icon = item.icon;
            return (
              <button
                key={item.key}
                className={`rail-button ${activeView === item.key ? 'active' : ''}`}
                onClick={() => onNavigate(item.key)}
                title={item.label}
                aria-label={item.label}
              >
                <Icon size={20} />
              </button>
            );
          })}
        </nav>
        <div className="rail-covers">
          {playlist.slice(0, 7).map((track) => (
            <button key={track.id} className="rail-cover" onClick={() => void player.play(track, playlist)} title={track.title}>
              <img src={track.thumbnail} alt="" />
            </button>
          ))}
        </div>
          <span />
      </aside>

      <main className="content frame">
        <span className="frame-label">Main</span>
        {children}
      </main>

      <aside className="right-panel frame">
        <span className="frame-label">Sidebar</span>
        {player.currentTrack ? (
          <div className="now-playing-panel">
            <div className="record-art">
              <img src={player.currentTrack.thumbnail} alt="" />
              <span />
            </div>
            <div className="now-playing-meta">
              <strong>{player.currentTrack.title}</strong>
              <small>{player.currentTrack.uploader || 'Unknown Artist'}</small>
            </div>
            <button className="outline-action" onClick={() => onNavigate('lyrics')}>
              Show lyrics
            </button>
          </div>
        ) : (
          <div className="empty-side">
            <UserRound size={32} />
            <p>No track selected</p>
          </div>
        )}
        {activeView !== 'lyrics' && player.currentTrack && (
          <div className="side-lyrics">
            <LyricsPanel compact />
          </div>
        )}
      </aside>

      <PlayerBar />
    </div>
  );
}
