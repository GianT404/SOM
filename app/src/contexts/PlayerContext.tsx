import React, { createContext, useContext, useState, useCallback, useRef, useEffect } from 'react';
import { Audio } from 'expo-av';
import { SearchResult } from '../services/api';
import api from '../services/api';
import * as FileSystem from 'expo-file-system/legacy';
import { getPlaylist, OfflineTrack } from '../services/playlistStore';
import { getAudioSettings, AudioSettings, getBufferSamples, getSampleRateHz } from '../services/audioSettings';

interface Track {
    id: string;
    title: string;
    thumbnail: string;
    duration: number;
    uploader: string;
    localUri?: string; // local file path for offline playback
    lyrics?: any;      // offline saved lyrics
}

type RepeatMode = 'off' | 'all' | 'one';

interface PlayerState {
    currentTrack: Track | null;
    isPlaying: boolean;
    position: number;    // milliseconds
    duration: number;    // milliseconds
    isLoading: boolean;
    isShuffle: boolean;
    repeatMode: RepeatMode;
}

interface PlayerContextType extends PlayerState {
    play: (track: Track) => Promise<void>;
    pause: () => Promise<void>;
    resume: () => Promise<void>;
    seekTo: (position: number) => Promise<void>;
    stop: () => Promise<void>;
    skipToNext: () => Promise<void>;
    skipToPrevious: () => Promise<void>;
    toggleShuffle: () => void;
    toggleRepeat: () => void;
}

const PlayerContext = createContext<PlayerContextType | null>(null);

export const usePlayer = () => {
    const ctx = useContext(PlayerContext);
    if (!ctx) throw new Error('usePlayer must be inside PlayerProvider');
    return ctx;
};

export const PlayerProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const soundRef = useRef<Audio.Sound | null>(null);
    const currentPlayIdRef = useRef<number>(0);
    const [state, setState] = useState<PlayerState>({
        currentTrack: null,
        isPlaying: false,
        position: 0,
        duration: 0,
        isLoading: false,
        isShuffle: false,
        repeatMode: 'off',
    });

    // Refs to access latest state in callbacks without stale closures
    const stateRef = useRef(state);
    stateRef.current = state;

    // Keep a cached copy of the playlist for skip operations
    const playlistCacheRef = useRef<OfflineTrack[]>([]);

    // Audio engine settings (loaded from AsyncStorage)
    const audioSettingsRef = useRef<AudioSettings>({ bufferSize: 'balanced', sampleRate: 'auto' });

    // Refresh playlist cache periodically and on track change
    const refreshPlaylistCache = useCallback(async () => {
        try {
            const list = await getPlaylist();
            playlistCacheRef.current = list;
        } catch (e) {
            console.warn('[PlayerContext] Failed to refresh playlist cache:', e);
        }
    }, []);

    useEffect(() => {
        Audio.setAudioModeAsync({
            allowsRecordingIOS: false,
            staysActiveInBackground: true,
            playsInSilentModeIOS: true,
        });

        // Load audio engine settings
        getAudioSettings().then(settings => {
            audioSettingsRef.current = settings;
            console.log('[PlayerContext] Audio settings loaded:', settings);
        });

        // Initial cache load
        refreshPlaylistCache();

        return () => {
            soundRef.current?.unloadAsync();
        };
    }, [refreshPlaylistCache]);

    // Refresh cache when current track changes
    useEffect(() => {
        refreshPlaylistCache();
    }, [state.currentTrack?.id, refreshPlaylistCache]);

    // --- Get next track based on current playlist, shuffle, and repeat ---
    const getNextTrackFromQueue = useCallback((currentId: string): Track | null => {
        const list = playlistCacheRef.current;
        if (list.length === 0) return null;

        const currentIndex = list.findIndex(t => t.id === currentId);

        if (stateRef.current.isShuffle) {
            // Pick a random track that's not the current one
            if (list.length <= 1) return list[0] || null;
            let randomIndex: number;
            do {
                randomIndex = Math.floor(Math.random() * list.length);
            } while (randomIndex === currentIndex && list.length > 1);
            return list[randomIndex];
        }

        // Sequential: next track
        const nextIndex = currentIndex + 1;
        if (nextIndex >= list.length) {
            // End of list
            if (stateRef.current.repeatMode === 'all') {
                return list[0]; // Loop back to first
            }
            return null; // No more tracks
        }
        return list[nextIndex];
    }, []);

    const getPrevTrackFromQueue = useCallback((currentId: string): Track | null => {
        const list = playlistCacheRef.current;
        if (list.length === 0) return null;

        const currentIndex = list.findIndex(t => t.id === currentId);

        if (stateRef.current.isShuffle) {
            // Pick a random track that's not the current one
            if (list.length <= 1) return list[0] || null;
            let randomIndex: number;
            do {
                randomIndex = Math.floor(Math.random() * list.length);
            } while (randomIndex === currentIndex && list.length > 1);
            return list[randomIndex];
        }

        // Sequential: previous track
        const prevIndex = currentIndex - 1;
        if (prevIndex < 0) {
            if (stateRef.current.repeatMode === 'all') {
                return list[list.length - 1]; // Loop to last
            }
            return null;
        }
        return list[prevIndex];
    }, []);

    // --- Ref to hold playInternal for use in onPlaybackStatusUpdate ---
    const playInternalRef = useRef<(track: Track) => Promise<void>>(async () => { });

    // --- Handle track ending (uses refs to avoid stale closures) ---
    const handleTrackFinishedRef = useRef(async () => { });
    handleTrackFinishedRef.current = async () => {
        const current = stateRef.current;
        if (!current.currentTrack) return;

        // Repeat One: replay same track
        if (current.repeatMode === 'one') {
            await soundRef.current?.setPositionAsync(0);
            await soundRef.current?.playAsync();
            return;
        }

        // Try to get next track
        const nextTrack = getNextTrackFromQueue(current.currentTrack.id);
        if (nextTrack) {
            await playInternalRef.current(nextTrack);
        } else {
            // No more tracks, stop
            setState(prev => ({ ...prev, isPlaying: false }));
        }
    };

    // --- Playback status callback (stable, delegates via refs) ---
    const onPlaybackStatusUpdateRef = useRef((status: any) => { });
    onPlaybackStatusUpdateRef.current = (status: any) => {
        if (status.isLoaded) {
            setState(prev => ({
                ...prev,
                isPlaying: status.isPlaying,
                position: status.positionMillis || 0,
                duration: status.durationMillis || 0,
                isLoading: status.isBuffering,
            }));

            // Auto-advance when track finishes
            if (status.didJustFinish && !status.isLooping) {
                handleTrackFinishedRef.current();
            }
        }
    };

    // Stable callback function that never changes identity
    const onPlaybackStatusUpdate = useCallback((status: any) => {
        onPlaybackStatusUpdateRef.current(status);
    }, []);

    // --- Internal play (used by play, skipToNext, skipToPrevious) ---
    const playInternal = useCallback(async (track: Track) => {
        try {
            const playId = ++currentPlayIdRef.current;

            // Unload previous sound.
            if (soundRef.current) {
                await soundRef.current.unloadAsync();
                soundRef.current = null;
            }

            setState(prev => ({ ...prev, currentTrack: track, isLoading: true, isPlaying: false }));

            // Use local downloaded file if available, otherwise cache to temp file
            let audioUri = track.localUri;
            if (!audioUri) {
                const tempUri = `${FileSystem.cacheDirectory}temp_${track.id}.m4a`;
                const fileInfo = await FileSystem.getInfoAsync(tempUri);
                if (!fileInfo.exists) {
                    const streamUrl = api.getStreamUrl(track.id);
                    await FileSystem.downloadAsync(streamUrl, tempUri);
                }
                audioUri = tempUri;
            }

            // Ensure play wasn't cancelled or superseded during download
            if (currentPlayIdRef.current !== playId) {
                return;
            }

            // Reload audio engine settings from AsyncStorage before each play
            const settings = await getAudioSettings();
            audioSettingsRef.current = settings;

            const { sound } = await Audio.Sound.createAsync(
                { uri: audioUri },
                {
                    shouldPlay: true,
                    progressUpdateIntervalMillis: 250,
                    androidImplementation: 'MediaPlayer',
                },
                onPlaybackStatusUpdate
            );

            if (currentPlayIdRef.current !== playId) {
                await sound.unloadAsync();
                return;
            }

            soundRef.current = sound;
        } catch (error) {
            console.error('Play error:', error);
            setState(prev => ({ ...prev, isLoading: false }));
        }
    }, [onPlaybackStatusUpdate]);

    // Keep playInternalRef in sync
    playInternalRef.current = playInternal;

    // --- Public API ---
    const play = useCallback(async (track: Track) => {
        await playInternal(track);
    }, [playInternal]);

    const pause = useCallback(async () => {
        await soundRef.current?.pauseAsync();
    }, []);

    const resume = useCallback(async () => {
        await soundRef.current?.playAsync();
    }, []);

    const seekTo = useCallback(async (position: number) => {
        await soundRef.current?.setPositionAsync(position);
    }, []);

    const stop = useCallback(async () => {
        await soundRef.current?.stopAsync();
        await soundRef.current?.unloadAsync();
        soundRef.current = null;
        setState(prev => ({
            ...prev,
            currentTrack: null,
            isPlaying: false,
            position: 0,
            duration: 0,
            isLoading: false,
        }));
    }, []);

    const skipToNext = useCallback(async () => {
        const current = stateRef.current;
        if (!current.currentTrack) return;

        const nextTrack = getNextTrackFromQueue(current.currentTrack.id);
        if (nextTrack) {
            await playInternal(nextTrack);
        }
    }, [getNextTrackFromQueue, playInternal]);

    const skipToPrevious = useCallback(async () => {
        const current = stateRef.current;
        if (!current.currentTrack) return;

        // If we're more than 3 seconds in, restart the current track instead
        if (current.position > 3000) {
            await soundRef.current?.setPositionAsync(0);
            return;
        }

        const prevTrack = getPrevTrackFromQueue(current.currentTrack.id);
        if (prevTrack) {
            await playInternal(prevTrack);
        } else {
            // No previous track, restart current
            await soundRef.current?.setPositionAsync(0);
        }
    }, [getPrevTrackFromQueue, playInternal]);

    const toggleShuffle = useCallback(() => {
        setState(prev => ({ ...prev, isShuffle: !prev.isShuffle }));
    }, []);

    const toggleRepeat = useCallback(() => {
        setState(prev => {
            const modes: RepeatMode[] = ['off', 'all', 'one'];
            const currentIdx = modes.indexOf(prev.repeatMode);
            const nextMode = modes[(currentIdx + 1) % modes.length];
            return { ...prev, repeatMode: nextMode };
        });
    }, []);

    return (
        <PlayerContext.Provider value={{
            ...state,
            play, pause, resume, seekTo, stop,
            skipToNext, skipToPrevious, toggleShuffle, toggleRepeat,
        }}>
            {children}
        </PlayerContext.Provider>
    );
};
