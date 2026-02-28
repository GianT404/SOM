import React, { createContext, useContext, useState, useCallback, useRef, useEffect } from 'react';
import { Audio } from 'expo-av';
import { SearchResult } from '../services/api';
import api from '../services/api';
import * as FileSystem from 'expo-file-system/legacy';

interface Track {
    id: string;
    title: string;
    thumbnail: string;
    duration: number;
    uploader: string;
    localUri?: string; // local file path for offline playback
    lyrics?: any;      // offline saved lyrics
}

interface PlayerState {
    currentTrack: Track | null;
    isPlaying: boolean;
    position: number;    // milliseconds
    duration: number;    // milliseconds
    isLoading: boolean;
}

interface PlayerContextType extends PlayerState {
    play: (track: Track) => Promise<void>;
    pause: () => Promise<void>;
    resume: () => Promise<void>;
    seekTo: (position: number) => Promise<void>;
    stop: () => Promise<void>;
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
    });

    useEffect(() => {
        Audio.setAudioModeAsync({
            allowsRecordingIOS: false,
            staysActiveInBackground: true,
            playsInSilentModeIOS: true,
        });

        return () => {
            soundRef.current?.unloadAsync();
        };
    }, []);

    const onPlaybackStatusUpdate = useCallback((status: any) => {
        if (status.isLoaded) {
            setState(prev => ({
                ...prev,
                isPlaying: status.isPlaying,
                position: status.positionMillis || 0,
                duration: status.durationMillis || 0,
                isLoading: status.isBuffering,
            }));
        }
    }, []);

    const play = useCallback(async (track: Track) => {
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

            const { sound } = await Audio.Sound.createAsync(
                { uri: audioUri },
                { shouldPlay: true },
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
        setState({
            currentTrack: null,
            isPlaying: false,
            position: 0,
            duration: 0,
            isLoading: false,
        });
    }, []);

    return (
        <PlayerContext.Provider value={{ ...state, play, pause, resume, seekTo, stop }}>
            {children}
        </PlayerContext.Provider>
    );
};
