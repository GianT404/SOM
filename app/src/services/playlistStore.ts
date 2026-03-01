import AsyncStorage from '@react-native-async-storage/async-storage';
import * as FileSystem from 'expo-file-system/legacy';
import api from './api';

const PLAYLIST_KEY = 'dm4a_offline_playlist';
const DELETED_PLAYLIST_KEY = 'dm4a_deleted_playlist';
const DOWNLOAD_DIR = `${FileSystem.documentDirectory}dm4a_downloads/`;

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

    const url = api.getStreamUrl(track.id);
    const localUri = `${DOWNLOAD_DIR}${track.id}.m4a`;

    let lyricsData = null;
    try {
        lyricsData = await api.getLyrics(track.id);
    } catch (e) {
        console.warn("No lyrics found for", track.id);
    }

    // Download with progress
    const downloadResumable = FileSystem.createDownloadResumable(
        url,
        localUri,
        {},
        (downloadProgress: any) => {
            onProgress?.({
                totalBytesWritten: downloadProgress.totalBytesWritten,
                totalBytesExpectedToWrite: downloadProgress.totalBytesExpectedToWrite,
            });
        }
    );

    const result = await downloadResumable.downloadAsync();
    if (!result || !result.uri) {
        throw new Error('Download failed');
    }

    // Save to playlist
    const offlineTrack: OfflineTrack = {
        id: track.id,
        title: track.title,
        uploader: track.uploader,
        thumbnail: track.thumbnail,
        duration: track.duration,
        localUri: result.uri,
        downloadedAt: Date.now(),
        lyrics: lyricsData || undefined,
    };

    const updated = [...existing, offlineTrack];
    await savePlaylist(updated);

    return offlineTrack;
};
