import { invoke } from '@tauri-apps/api/core';
import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { api } from '../lib/api';
import { getPlaylist } from '../lib/storage';
import type { Track } from '../lib/types';

type RepeatMode = 'off' | 'all' | 'one';

interface PlayerState {
  currentTrack: Track | null;
  isPlaying: boolean;
  isLoading: boolean;
  position: number;
  duration: number;
  volume: number;
  isShuffle: boolean;
  repeatMode: RepeatMode;
  queue: Track[];
}

interface PlayerContextValue extends PlayerState {
  play: (track: Track, queue?: Track[]) => Promise<void>;
  pause: () => void;
  resume: () => void;
  togglePlay: () => void;
  seekTo: (positionMs: number) => void;
  setVolume: (volume: number) => void;
  skipToNext: () => void;
  skipToPrevious: () => void;
  toggleShuffle: () => void;
  toggleRepeat: () => void;
}

const PlayerContext = createContext<PlayerContextValue | null>(null);

export function usePlayer() {
  const ctx = useContext(PlayerContext);
  if (!ctx) throw new Error('usePlayer must be used inside PlayerProvider');
  return ctx;
}

export function PlayerProvider({ children }: { children: React.ReactNode }) {
  const audioRef = useRef(new Audio());
  const localObjectUrlRef = useRef<string | null>(null);
  const stateRef = useRef<PlayerState | null>(null);
  const [state, setState] = useState<PlayerState>({
    currentTrack: null,
    isPlaying: false,
    isLoading: false,
    position: 0,
    duration: 0,
    volume: 0.9,
    isShuffle: false,
    repeatMode: 'off',
    queue: [],
  });

  stateRef.current = state;

  const play = useCallback(async (track: Track, queue?: Track[]) => {
    const audio = audioRef.current;

    setState((prev) => ({
      ...prev,
      currentTrack: track,
      queue: queue?.length ? queue : prev.queue.length ? prev.queue : [track],
      isLoading: true,
      position: 0,
    }));

    try {
      if (localObjectUrlRef.current) {
        URL.revokeObjectURL(localObjectUrlRef.current);
        localObjectUrlRef.current = null;
      }

      if (track.localPath) {
        try {
          const bytes = await invoke<number[]>('read_downloaded_track', { localPath: track.localPath });
          const blob = new Blob([new Uint8Array(bytes)], { type: 'audio/mp4' });
          localObjectUrlRef.current = URL.createObjectURL(blob);
          audio.src = localObjectUrlRef.current;
        } catch (error) {
          console.error('[Player] local playback failed, falling back to stream', error);
          audio.src = api.getStreamUrl(track.id);
        }
      } else {
        audio.src = api.getStreamUrl(track.id);
      }

      audio.volume = stateRef.current?.volume ?? 0.9;
      audio.load();
      await audio.play();
    } catch (error) {
      console.error('[Player] play failed', error);
      setState((prev) => ({ ...prev, isPlaying: false, isLoading: false }));
    }
  }, []);

  const pause = useCallback(() => audioRef.current.pause(), []);
  const resume = useCallback(() => void audioRef.current.play(), []);
  const togglePlay = useCallback(() => {
    if (audioRef.current.paused) resume();
    else pause();
  }, [pause, resume]);
  const seekTo = useCallback((positionMs: number) => {
    audioRef.current.currentTime = Math.max(0, positionMs / 1000);
  }, []);
  const setVolume = useCallback((volume: number) => {
    const next = Math.min(1, Math.max(0, volume));
    audioRef.current.volume = next;
    setState((prev) => ({ ...prev, volume: next }));
  }, []);

  const pickRelativeTrack = useCallback((direction: 1 | -1) => {
    const current = stateRef.current;
    if (!current?.currentTrack) return null;
    const queue = current.queue.length ? current.queue : getPlaylist();
    if (!queue.length) return null;
    if (current.isShuffle && queue.length > 1) {
      const choices = queue.filter((track) => track.id !== current.currentTrack?.id);
      return choices[Math.floor(Math.random() * choices.length)] ?? null;
    }
    const index = queue.findIndex((track) => track.id === current.currentTrack?.id);
    const nextIndex = index + direction;
    if (nextIndex >= queue.length) return current.repeatMode === 'all' ? queue[0] : null;
    if (nextIndex < 0) return current.repeatMode === 'all' ? queue[queue.length - 1] : null;
    return queue[nextIndex] ?? null;
  }, []);

  const skipToNext = useCallback(() => {
    const current = stateRef.current;
    if (current?.repeatMode === 'one' && current.currentTrack) {
      void play(current.currentTrack, current.queue);
      return;
    }
    const next = pickRelativeTrack(1);
    if (next) void play(next, stateRef.current?.queue);
  }, [pickRelativeTrack, play]);

  const skipToPrevious = useCallback(() => {
    const current = stateRef.current;
    if ((current?.position ?? 0) > 3000) {
      seekTo(0);
      return;
    }
    const previous = pickRelativeTrack(-1);
    if (previous) void play(previous, stateRef.current?.queue);
  }, [pickRelativeTrack, play, seekTo]);

  useEffect(() => {
    const audio = audioRef.current;
    const onLoadStart = () => setState((prev) => ({ ...prev, isLoading: true }));
    const onCanPlay = () => setState((prev) => ({ ...prev, isLoading: false }));
    const onPlay = () => setState((prev) => ({ ...prev, isPlaying: true, isLoading: false }));
    const onPause = () => setState((prev) => ({ ...prev, isPlaying: false }));
    const onTimeUpdate = () =>
      setState((prev) => ({
        ...prev,
        position: audio.currentTime * 1000,
        duration: Number.isFinite(audio.duration) ? audio.duration * 1000 : prev.duration,
      }));
    const onEnded = () => skipToNext();

    audio.addEventListener('loadstart', onLoadStart);
    audio.addEventListener('canplay', onCanPlay);
    audio.addEventListener('play', onPlay);
    audio.addEventListener('pause', onPause);
    audio.addEventListener('timeupdate', onTimeUpdate);
    audio.addEventListener('durationchange', onTimeUpdate);
    audio.addEventListener('ended', onEnded);

    return () => {
      audio.pause();
      if (localObjectUrlRef.current) {
        URL.revokeObjectURL(localObjectUrlRef.current);
        localObjectUrlRef.current = null;
      }
      audio.removeEventListener('loadstart', onLoadStart);
      audio.removeEventListener('canplay', onCanPlay);
      audio.removeEventListener('play', onPlay);
      audio.removeEventListener('pause', onPause);
      audio.removeEventListener('timeupdate', onTimeUpdate);
      audio.removeEventListener('durationchange', onTimeUpdate);
      audio.removeEventListener('ended', onEnded);
    };
  }, [skipToNext]);

  const value = useMemo<PlayerContextValue>(() => ({
    ...state,
    play,
    pause,
    resume,
    togglePlay,
    seekTo,
    setVolume,
    skipToNext,
    skipToPrevious,
    toggleShuffle: () => setState((prev) => ({ ...prev, isShuffle: !prev.isShuffle })),
    toggleRepeat: () =>
      setState((prev) => {
        const modes: RepeatMode[] = ['off', 'all', 'one'];
        return { ...prev, repeatMode: modes[(modes.indexOf(prev.repeatMode) + 1) % modes.length] };
      }),
  }), [pause, play, resume, seekTo, setVolume, skipToNext, skipToPrevious, state, togglePlay]);

  return <PlayerContext.Provider value={value}>{children}</PlayerContext.Provider>;
}
