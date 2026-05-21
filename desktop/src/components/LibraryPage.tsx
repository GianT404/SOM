import { FolderDown } from 'lucide-react';
import { useEffect, useState } from 'react';
import { getPlaylist, permanentlyDeleteTrack, softDeleteTrack } from '../lib/storage';
import type { OfflineTrack } from '../lib/types';
import { usePlayer } from '../stores/playerStore';
import { TrackTable } from './TrackTable';

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
      {tracks.length === 0 ? (
        <div className="empty-state">
          <FolderDown size={52} />
          <h2>No offline tracks yet</h2>
          <p>Download songs from the player menu to listen offline.</p>
        </div>
      ) : (
        <TrackTable
          tracks={tracks}
          onPlay={(track) => void play(track, tracks)}
          sourceLabel={() => 'SOM Downloads'}
          onDelete={(id) => {
            softDeleteTrack(id);
            permanentlyDeleteTrack(id);
          }}
        />
      )}
    </section>
  );
}
