import { RotateCcw, Trash2 } from 'lucide-react';
import type { Track } from '../lib/types';
import { formatDuration } from './format';

interface TrackTableProps<T extends Track> {
  tracks: T[];
  currentId?: string;
  onPlay: (track: T) => void;
  sourceLabel?: (track: T) => string;
  onDelete?: (id: string) => void;
  onRestore?: (id: string) => void;
}

export function TrackTable<T extends Track>({
  tracks,
  currentId,
  onPlay,
  sourceLabel,
  onDelete,
  onRestore,
}: TrackTableProps<T>) {
  return (
    <div className="track-table" role="table">
      <div className="track-table-head" role="row">
        <span>#</span>
        <span>Title</span>
        <span>Album</span>
        <span>Time</span>
        <span />
      </div>
      {tracks.map((track, index) => {
        const isActive = currentId === track.id;
        return (
          <div className={`track-table-row ${isActive ? 'active' : ''}`} role="row" key={track.id}>
            <button className="track-index" onClick={() => onPlay(track)} aria-label={`Play ${track.title}`}>
              {isActive ? <span className="playing-bars">▥</span> : index + 1}
            </button>
            <button className="track-title-cell" onClick={() => onPlay(track)}>
              <img src={track.thumbnail} alt="" />
              <span>
                <strong>{track.title}</strong>
                <small>{track.uploader || 'Unknown Artist'}</small>
              </span>
            </button>
            <button className="track-source-cell" onClick={() => onPlay(track)}>
              {sourceLabel?.(track) ?? track.uploader ?? 'YouTube'}
            </button>
            <button className="track-time-cell" onClick={() => onPlay(track)}>
              {formatDuration(track.duration * 1000)}
            </button>
            <span className="track-action-cell">
              {onDelete && (
                <button className="small-icon danger" onClick={() => onDelete(track.id)} aria-label="Delete track">
                  <Trash2 size={15} />
                </button>
              )}
              {onRestore && (
                <button className="small-icon" onClick={() => onRestore(track.id)} aria-label="Restore track">
                  <RotateCcw size={15} />
                </button>
              )}
            </span>
          </div>
        );
      })}
    </div>
  );
}
