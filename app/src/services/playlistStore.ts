import AsyncStorage from '@react-native-async-storage/async-storage';
import * as FileSystem from 'expo-file-system/legacy';
import api from './api';

const PLAYLIST_KEY = 'dm4a_offline_playlist';
const DELETED_PLAYLIST_KEY = 'dm4a_deleted_playlist';
const DOWNLOAD_DIR = `${FileSystem.documentDirectory}dm4a_downloads/`;

export const saveCustomLyrics = async (trackId: string, customLyrics: string): Promise<boolean> => {
    try {
        const data = await AsyncStorage.getItem(PLAYLIST_KEY);
        if (!data) return false;

        const playlist = JSON.parse(data);
        const trackIndex = playlist.findIndex((t: any) => t.id === trackId);

        if (trackIndex !== -1) {
            // Ghi đè lời bài hát thủ công vào object track
            playlist[trackIndex].lyrics = customLyrics;
            await AsyncStorage.setItem(PLAYLIST_KEY, JSON.stringify(playlist));
            return true;
        }
        return false;
    } catch (error) {
        console.error('[Store] Lỗi khi lưu lời bài hát thủ công:', error);
        return false;
    }
};
export interface OfflineTrack {
    id: string;
    title: string;
    uploader: string;
    thumbnail: string;
    duration: number;
    localUri: string;
    downloadedAt: number;
    lyrics?: import('./api').LyricsData[];
}

// Ensure download directory exists
const ensureDir = async () => {
    const info = await FileSystem.getInfoAsync(DOWNLOAD_DIR);
    if (!info.exists) {
        await FileSystem.makeDirectoryAsync(DOWNLOAD_DIR, { intermediates: true });
    }
};

// ---------- Playlist CRUD ----------

export const getPlaylist = async (): Promise<OfflineTrack[]> => {
    try {
        const json = await AsyncStorage.getItem(PLAYLIST_KEY);
        return json ? JSON.parse(json) : [];
    } catch {
        return [];
    }
};

const savePlaylist = async (tracks: OfflineTrack[]) => {
    await AsyncStorage.setItem(PLAYLIST_KEY, JSON.stringify(tracks));
};

export const isDownloaded = async (id: string): Promise<boolean> => {
    const list = await getPlaylist();
    return list.some(t => t.id === id);
};

export const getDeletedPlaylist = async (): Promise<OfflineTrack[]> => {
    try {
        const json = await AsyncStorage.getItem(DELETED_PLAYLIST_KEY);
        return json ? JSON.parse(json) : [];
    } catch {
        return [];
    }
};

const saveDeletedPlaylist = async (tracks: OfflineTrack[]) => {
    await AsyncStorage.setItem(DELETED_PLAYLIST_KEY, JSON.stringify(tracks));
};

export const softDeleteTrack = async (id: string) => {
    const list = await getPlaylist();
    const track = list.find(t => t.id === id);
    if (track) {
        // Remove from main playlist
        await savePlaylist(list.filter(t => t.id !== id));

        // Add to deleted playlist
        const deletedList = await getDeletedPlaylist();
        await saveDeletedPlaylist([...deletedList, track]);
    }
};

export const restoreTrack = async (id: string) => {
    const deletedList = await getDeletedPlaylist();
    const track = deletedList.find(t => t.id === id);
    if (track) {
        // Remove from deleted playlist
        await saveDeletedPlaylist(deletedList.filter(t => t.id !== id));

        // Add to main playlist
        const list = await getPlaylist();
        await savePlaylist([...list, track]);
    }
};

//add removeFromPlaylist
export const removeFromPlaylist = async (id: string) => {
    const list = await getPlaylist();
    await savePlaylist(list.filter(t => t.id !== id));
};


export const permanentlyDeleteTrack = async (id: string) => {
    const list = await getDeletedPlaylist();
    const track = list.find(t => t.id === id);
    if (track) {
        // Delete local file
        try { await FileSystem.deleteAsync(track.localUri, { idempotent: true }); } catch { }
    }
    await saveDeletedPlaylist(list.filter(t => t.id !== id));
};

// ---------- Download + Add ----------

export interface DownloadProgress {
    totalBytesWritten: number;
    totalBytesExpectedToWrite: number;
}

export const downloadAndAdd = async (
    track: { id: string; title: string; uploader: string; thumbnail: string; duration: number },
    onProgress?: (progress: DownloadProgress) => void,
): Promise<OfflineTrack> => {
    await ensureDir();

    // Check if already downloaded
    const existing = await getPlaylist();
    const found = existing.find(t => t.id === track.id);
    if (found) return found;

    const localUri = `${DOWNLOAD_DIR}${track.id}.m4a`;

    // Signal "preparing" while server runs yt-dlp (before any bytes flow)
    onProgress?.({ totalBytesWritten: 0, totalBytesExpectedToWrite: -1 });

    // Fetch lyrics in parallel with the download (pass metadata for LRCLib)
    const lyricsPromise = api.getLyrics(track.id, {
        title: track.title,
        artist: track.uploader,
        duration: track.duration,
    }).catch((e) => {
        console.warn('[downloadAndAdd] No lyrics:', e?.message);
        return null;
    });

    // Download via server /stream endpoint — yt-dlp handles n-parameter throttle
    // decryption internally, so we get full CDN speed (vs ~30KB/s on raw URL).
    const streamUrl = api.getStreamUrl(track.id);
    console.log('[downloadAndAdd] Downloading via /stream for', track.id);

    const downloadResumable = FileSystem.createDownloadResumable(
        streamUrl,
        localUri,
        {},
        (downloadProgress: any) => {
            const written = downloadProgress.totalBytesWritten;
            const total = downloadProgress.totalBytesExpectedToWrite;
            console.log('[downloadAndAdd] Progress:', written, '/', total);
            onProgress?.({ totalBytesWritten: written, totalBytesExpectedToWrite: total });
        }
    );

    const result = await downloadResumable.downloadAsync();
    if (!result || !result.uri) {
        throw new Error('Download failed');
    }

    // Save track immediately so it appears in playlist right away.
    // We do NOT wait for lyrics here — they will be patched in the background.
    const offlineTrack: OfflineTrack = {
        id: track.id,
        title: track.title,
        uploader: track.uploader,
        thumbnail: track.thumbnail,
        duration: track.duration,
        localUri: result.uri,
        downloadedAt: Date.now(),
        lyrics: undefined, // filled in below once lyrics resolve
    };

    const updated = [...existing, offlineTrack];
    await savePlaylist(updated);

    // Patch lyrics in the background — doesn't block the caller.
    lyricsPromise.then(async (lyricsData) => {
        if (!lyricsData) return;
        try {
            const current = await getPlaylist();
            const patched = current.map(t =>
                t.id === track.id ? { ...t, lyrics: lyricsData } : t
            );
            await savePlaylist(patched);
            console.log('[downloadAndAdd] Lyrics patched for', track.id);
        } catch (e) {
            console.warn('[downloadAndAdd] Failed to patch lyrics:', e);
        }
    });

    return offlineTrack;
};

// ---------- Navigation & Logic ----------

/**
 * Lấy bài hát tiếp theo trong danh sách
 */
export const getNextTrack = async (currentId: string): Promise<OfflineTrack | null> => {
    const list = await getPlaylist();
    if (list.length === 0) return null;

    const currentIndex = list.findIndex(t => t.id === currentId);
    // Nếu không tìm thấy hoặc là bài cuối cùng -> quay về bài đầu (Loop Queue)
    const nextIndex = (currentIndex + 1) % list.length;
    return list[nextIndex];
};

/**
 * Lấy bài hát trước đó
 */
export const getPreviousTrack = async (currentId: string): Promise<OfflineTrack | null> => {
    const list = await getPlaylist();
    if (list.length === 0) return null;

    const currentIndex = list.findIndex(t => t.id === currentId);
    // Nếu là bài đầu tiên -> nhảy xuống bài cuối cùng
    const prevIndex = (currentIndex - 1 + list.length) % list.length;
    return list[prevIndex];
};

/**
 * Trộn danh sách bài hát (Shuffle)
 */
export const shuffleTracks = (list: OfflineTrack[]): OfflineTrack[] => {
    const shuffled = [...list];
    for (let i = shuffled.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
    }
    return shuffled;
};