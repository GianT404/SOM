import { Clock3, Search } from 'lucide-react';
import { useEffect, useState } from 'react';
import type { OfflineTrack, ViewKey } from '../lib/types';
import { getDeletedPlaylist, getPlaylist, restoreTrack, softDeleteTrack } from '../lib/storage';
import { usePlayer } from '../stores/playerStore';
import { TrackTable } from './TrackTable';

export function HomePage({ onNavigate }: { onNavigate: (view: ViewKey) => void }) {
  const { play, currentTrack } = usePlayer();
  const [tracks, setTracks] = useState<OfflineTrack[]>([]);
  const [deletedTracks, setDeletedTracks] = useState<OfflineTrack[]>([]);
  const [tab, setTab] = useState<'playlist' | 'deleted'>('playlist');

  useEffect(() => {
    const load = () => {
      setTracks(getPlaylist());
      setDeletedTracks(getDeletedPlaylist());
    };
    load();
    window.addEventListener('som-playlist-changed', load);
    return () => window.removeEventListener('som-playlist-changed', load);
  }, []);

  const visible = tab === 'playlist' ? tracks : deletedTracks;

  return (
    <section className="page home-page">
      <div className="tab-row">
        <button className={tab === 'playlist' ? 'active' : ''} onClick={() => setTab('playlist')}>
          Songs <span>{tracks.length}</span>
        </button>
        <button className={tab === 'deleted' ? 'active' : ''} onClick={() => setTab('deleted')}>
          Đã xóa
        </button>
      </div>

      {visible.length === 0 ? (
        <div className="empty-state">
          <Clock3 size={46} />
          <h2>{tab === 'playlist' ? 'Trống như đường tình bạn vậy?' : 'Thùng rác trống'}</h2>
          <p>{tab === 'playlist' ? 'Tìm kiếm, thêm nhạc, chill thôi nào.' : 'Không có bài hát nào bị xóa.'}</p>
        </div>
      ) : (
        <TrackTable
          tracks={visible}
          currentId={currentTrack?.id}
          onPlay={(track) => void play(track, tracks)}
          sourceLabel={() => (tab === 'playlist' ? 'SOM Downloads' : 'Recently Deleted')}
          onDelete={tab === 'playlist' ? softDeleteTrack : undefined}
          onRestore={tab === 'deleted' ? restoreTrack : undefined}
        />
      )}
    </section>
  );
}
