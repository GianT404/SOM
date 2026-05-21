import { invoke } from '@tauri-apps/api/core';
import { Download, Loader2, Pause, Play, Repeat, Repeat1, Shuffle, SkipBack, SkipForward, Volume2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../lib/api';
import { getPlaylist, upsertPlaylistTrack } from '../lib/storage';
import type { LyricsData, OfflineTrack, Track } from '../lib/types';
import { usePlayer } from '../stores/playerStore';
import { formatDuration } from './format';

interface DownloadResponse {
  id: string;
  title: string;
  uploader: string;
  thumbnail: string;
  duration: number;
  local_path: string;
  downloaded_at: number;
  lyrics?: LyricsData[];
}

export function PlayerBar() {
  const player = usePlayer();
  const [downloadState, setDownloadState] = useState<'idle' | 'downloading' | 'done'>('idle');
  const [downloadError, setDownloadError] = useState('');
  const track = player.currentTrack;
  const progress = player.duration > 0 ? player.position / player.duration : 0;

  useEffect(() => {
    if (!track) return;
    setDownloadState(getPlaylist().some((item) => item.id === track.id) ? 'done' : 'idle');
    setDownloadError('');
  }, [track?.id, track]);

  useEffect(() => {
    const onSpace = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      const isTyping = target?.tagName === 'INPUT' || target?.tagName === 'TEXTAREA';
      if (!isTyping && event.code === 'Space') {
        event.preventDefault();
        player.togglePlay();
      }
    };
    const onNext = () => player.skipToNext();
    const onPrevious = () => player.skipToPrevious();
    window.addEventListener('keydown', onSpace);
    window.addEventListener('som-next', onNext);
    window.addEventListener('som-previous', onPrevious);
    return () => {
      window.removeEventListener('keydown', onSpace);
      window.removeEventListener('som-next', onNext);
      window.removeEventListener('som-previous', onPrevious);
    };
  }, [player]);

  const download = async () => {
    if (!track || downloadState !== 'idle') return;
    setDownloadState('downloading');
    setDownloadError('');
    try {
      const lyrics = await api.getLyrics(track.id, {
        title: track.title,
        artist: track.uploader,
        duration: track.duration,
      }).catch(() => undefined);
      const response = await invoke<DownloadResponse>('download_track', {
        track: {
          id: track.id,
          title: track.title,
          uploader: track.uploader,
          thumbnail: track.thumbnail,
          duration: track.duration,
          stream_url: api.getStreamUrl(track.id),
          lyrics,
        },
      });
      const offlineTrack: OfflineTrack = normalizeDownload(response, track);
      upsertPlaylistTrack(offlineTrack);
      setDownloadState('done');
    } catch (error) {
      setDownloadState('idle');
      setDownloadError(error instanceof Error ? error.message : String(error));
    }
  };

  return (
    <footer className="player-bar frame">
      <span className="frame-label">Playing</span>
      <div className={`player-track ${track ? '' : 'empty'}`}>
        {track ? (
          <>
            <div>
              <strong>{track.title}</strong>
              <small>{track.uploader || 'Không bít'}</small>
            </div>
          </>
        ) : (
          <div>
            <strong>No track playing</strong>
            <small>Select a song from the list</small>
          </div>
        )}
      </div>
      <div className="player-center">
        <div className="control-row">
          <button className={player.isShuffle ? 'active icon-button' : 'icon-button'} onClick={player.toggleShuffle} disabled={!track} aria-label="Shuffle">
            <Shuffle size={18} />
          </button>
          <button className="icon-button" onClick={player.skipToPrevious} disabled={!track} aria-label="Previous">
            <SkipBack size={22} />
          </button>
          <button className="play-button" onClick={player.togglePlay} disabled={!track} aria-label={player.isPlaying ? 'Pause' : 'Play'}>
            {player.isLoading ? <Loader2 className="spin" size={24} /> : player.isPlaying ? <Pause size={26} /> : <Play size={26} />}
          </button>
          <button className="icon-button" onClick={player.skipToNext} disabled={!track} aria-label="Next">
            <SkipForward size={22} />
          </button>
          <button className={player.repeatMode !== 'off' ? 'active icon-button' : 'icon-button'} onClick={player.toggleRepeat} disabled={!track} aria-label="Repeat">
            {player.repeatMode === 'one' ? <Repeat1 size={18} /> : <Repeat size={18} />}
          </button>
        </div>
        <div className="seek-row">
          <span>{formatDuration(player.position)}</span>
          <input
          type="range"
          min={0}
          max={player.duration || 0}
          value={player.position}
          disabled={!track}
          onChange={(event) => player.seekTo(Number(event.currentTarget.value))}
          style={{ ['--progress' as string]: `${progress * 100}%` }}
        />
          <span>{formatDuration(player.duration)}</span>
        </div>
      </div>
      <div className="player-actions">
        <button
          className={`icon-button ${downloadState === 'done' ? 'active' : ''}`}
          onClick={() => void download()}
          disabled={!track || downloadState !== 'idle'}
          title={downloadError || (downloadState === 'done' ? 'Downloaded' : 'Download')}
          aria-label="Download track"
        >
          {downloadState === 'downloading' ? <Loader2 className="spin" size={18} /> : <Download size={18} />}
        </button>
        <Volume2 size={18} />
        <input
          className="volume"
          type="range"
          min={0}
          max={1}
          step={0.01}
          value={player.volume}
          onChange={(event) => player.setVolume(Number(event.currentTarget.value))}
        />
      </div>
    </footer>
  );
}

function normalizeDownload(response: DownloadResponse, track: Track): OfflineTrack {
  return {
    id: response.id,
    title: response.title,
    uploader: response.uploader,
    thumbnail: response.thumbnail || track.thumbnail,
    duration: response.duration,
    localPath: response.local_path,
    downloadedAt: response.downloaded_at,
    lyrics: response.lyrics,
  };
}
