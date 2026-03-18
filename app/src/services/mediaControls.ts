/**
 * mediaControls.ts
 * 
 * Service quản lý hiển thị media controls trên notification bar và lock screen.
 * Sử dụng expo-media-control để tích hợp với Android MediaSession API.
 */
import {
    MediaControl,
    PlaybackState,
    Command,
    MediaControlEvent,
} from 'expo-media-control';

// --- Kiểu dữ liệu callback từ notification/lock screen ---
interface MediaControlCallbacks {
    onPlay: () => void;
    onPause: () => void;
    onStop: () => void;
    onNextTrack: () => void;
    onPreviousTrack: () => void;
    onSeek?: (positionMs: number) => void;
}

let listenerCleanup: (() => void) | null = null;

/**
 * Khởi tạo media controls với các nút: play, pause, stop, prev, next, seek.
 * Gọi 1 lần khi app khởi động.
 */
export async function initMediaControls(): Promise<void> {
    try {
        await MediaControl.enableMediaControls({
            capabilities: [
                Command.PLAY,
                Command.PAUSE,
                Command.STOP,
                Command.NEXT_TRACK,
                Command.PREVIOUS_TRACK,
                Command.SEEK,
            ],
            compactCapabilities: [
                Command.PREVIOUS_TRACK,
                Command.PLAY,
                Command.NEXT_TRACK,
            ],
            notification: {
                color: '#FFB86C',
            },
        });
        console.log('[MediaControls] Initialized successfully');
    } catch (error) {
        console.error('[MediaControls] Failed to initialize:', error);
    }
}

/**
 * Cập nhật metadata bài hát hiện tại (title, artist, thumbnail, duration).
 * Gọi mỗi khi track thay đổi.
 */
export async function updateNowPlaying(
    title: string,
    artist: string,
    artworkUri: string,
    durationSeconds: number
): Promise<void> {
    try {
        await MediaControl.updateMetadata({
            title,
            artist,
            artwork: { uri: artworkUri },
            duration: durationSeconds,
        });
    } catch (error) {
        console.error('[MediaControls] Failed to update metadata:', error);
    }
}

/**
 * Cập nhật trạng thái playback (playing/paused/stopped/buffering) và vị trí hiện tại.
 * Gọi trong onPlaybackStatusUpdate.
 */
export async function updatePlayback(
    state: 'playing' | 'paused' | 'stopped' | 'buffering',
    positionSeconds?: number
): Promise<void> {
    try {
        const stateMap: Record<string, PlaybackState> = {
            playing: PlaybackState.PLAYING,
            paused: PlaybackState.PAUSED,
            stopped: PlaybackState.STOPPED,
            buffering: PlaybackState.BUFFERING,
        };
        await MediaControl.updatePlaybackState(stateMap[state], positionSeconds);
    } catch (error) {
        console.error('[MediaControls] Failed to update playback state:', error);
    }
}

/**
 * Đăng ký listener cho các sự kiện từ notification/lock screen.
 * Trả về hàm cleanup để gỡ listener.
 */
export function setupEventListener(callbacks: MediaControlCallbacks): () => void {
    // Gỡ listener cũ nếu có
    if (listenerCleanup) {
        listenerCleanup();
        listenerCleanup = null;
    }

    const removeListener = MediaControl.addListener((event: MediaControlEvent) => {
        console.log('[MediaControls] Event:', event.command);
        switch (event.command) {
            case Command.PLAY:
                callbacks.onPlay();
                break;
            case Command.PAUSE:
                callbacks.onPause();
                break;
            case Command.STOP:
                callbacks.onStop();
                break;
            case Command.NEXT_TRACK:
                callbacks.onNextTrack();
                break;
            case Command.PREVIOUS_TRACK:
                callbacks.onPreviousTrack();
                break;
            case Command.SEEK:
                if (callbacks.onSeek && event.data?.position !== undefined) {
                    // expo-media-control trả position theo giây, convert sang ms
                    callbacks.onSeek(event.data.position * 1000);
                }
                break;
        }
    });

    listenerCleanup = removeListener;
    return removeListener;
}

/**
 * Dọn dẹp toàn bộ media controls.
 * Gọi khi app unmount hoặc khi cần reset.
 */
export async function cleanupMediaControls(): Promise<void> {
    try {
        if (listenerCleanup) {
            listenerCleanup();
            listenerCleanup = null;
        }
        await MediaControl.disableMediaControls();
        console.log('[MediaControls] Cleaned up');
    } catch (error) {
        console.error('[MediaControls] Failed to cleanup:', error);
    }
}
