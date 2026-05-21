import { FolderDown, Trash2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import { getPlaylist, permanentlyDeleteTrack, softDeleteTrack } from '../lib/storage';
import type { OfflineTrack } from '../lib/types';
import { usePlayer } from '../stores/playerStore';
import { SongRow } from './SongRow';

export function LibraryPage() {
  const { play } = usePlayer();
  const [tracks, setTracks] = useState<OfflineTrack[]>([]);

  useEffect(() => {
    const load = () => setTracks(getPlaylist());
    load();
    window.addEventListener('som-playlist-changed', load);
    return () => window.removeEventListener('som-playlist-changed', load);
  }, []);

  return (
    <section className="page">
      <header className="topbar">
        <div>
          <p className="eyebrow">{tracks.length} downloaded tracks</p>
          <h1>My Playlist</h1>
        </div>
      </header>
      {tracks.length === 0 ? (
        <div className="empty-state">
          <FolderDown size={52} />
          <h2>No offline tracks yet</h2>
          <p>Download songs from the player menu to listen offline.</p>
        </div>
      ) : (
        <div className="song-list">
          {tracks.map((track) => (
            <div className="library-row" key={track.id}>
              <SongRow track={track} onPlay={() => void play(track, tracks)} />
              <button
                className="square-button danger"
                onClick={() => {
                  softDeleteTrack(track.id);
                  permanentlyDeleteTrack(track.id);
                }}
                aria-label="Remove download"
              >
                <Trash2 size={18} />
              </button>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

