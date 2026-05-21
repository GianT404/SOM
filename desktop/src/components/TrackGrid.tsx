import { RotateCcw, Trash2 } from 'lucide-react';
import type { OfflineTrack } from '../lib/types';
import { formatDuration } from './format';

interface TrackGridProps {
  tracks: OfflineTrack[];
  currentId?: string;
  onPlay: (track: OfflineTrack) => void;
  onDelete?: (id: string) => void;
  onRestore?: (id: string) => void;
}

export function TrackGrid({ tracks, currentId, onPlay, onDelete, onRestore }: TrackGridProps) {
  return (
    <div className="track-grid">
      {tracks.map((track) => (
        <article key={track.id} className={`music-card neo-card shadow-lg ${currentId === track.id ? 'active' : ''}`}>
          <button className="music-card-main" onClick={() => onPlay(track)}>
            <img src={track.thumbnail} alt="" />
            <strong>{track.title}</strong>
            <small>{track.uploader}</small>
            <span>{formatDuration(track.duration * 1000)}</span>
          </button>
          {onDelete && (
            <button className="card-action danger" onClick={() => onDelete(track.id)} aria-label="Delete track">
              <Trash2 size={18} />
            </button>
          )}
          {onRestore && (
            <button className="card-action" onClick={() => onRestore(track.id)} aria-label="Restore track">
              <RotateCcw size={18} />
            </button>
          )}
        </article>
      ))}
    </div>
  );
}

