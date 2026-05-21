import { Play } from 'lucide-react';
import type { Track } from '../lib/types';
import { formatDuration } from './format';

export function SongRow({ track, onPlay }: { track: Track; onPlay: () => void }) {
  return (
    <button className="song-row neo-card shadow-sm" onClick={onPlay}>
      <img src={track.thumbnail} alt="" />
      <span className="song-row-meta">
        <strong>{track.title}</strong>
        <small>{track.uploader}</small>
      </span>
      <small>{formatDuration(track.duration * 1000)}</small>
      <span className="icon-pill">
        <Play size={18} />
      </span>
    </button>
  );
}

