import * as FileSystem from 'expo-file-system/legacy';
import {
    getPlaylist,
    upsertOfflineTrack,
    OfflineTrack,
} from './playlistStore';

export interface TransferManifestEntry {
    id: string;
    title: string;
    artist: string;
    duration: number;
    thumbnail?: string;
    filename: string;
    size: number;
    md5: string;
}

interface PairResponse {
    version: number;
    type: 'som-transfer';
    token: string;
}

const DOWNLOAD_DIR = `${FileSystem.documentDirectory}dm4a_downloads/`;

function extensionFor(filename: string): string {
    const match = filename.match(/(\.[a-z0-9]+)$/i);
    const ext = match?.[1]?.toLowerCase();
    return !ext || ext.length > 8 ? '.m4a' : ext;
}

function authHeaders(token: string): Record<string, string> {
    return {
        'X-SOM-Transfer-Token': token,
    };
}

function safeLocalName(id: string): string {
    return id.replace(/[^a-zA-Z0-9_-]/g, '_');
}

export class TransferClient {
    private baseUrl = '';
    private token = '';

    async pair(pairUrl: string): Promise<PairResponse> {
        const parsed = new URL(pairUrl.trim());
        if (!['http:', 'https:'].includes(parsed.protocol)) {
            throw new Error('Chỉ hỗ trợ http/https');
        }
        if (!parsed.searchParams.get('token')) {
            throw new Error('Pair URL không có token');
        }

        const res = await fetch(parsed.toString(), {
            headers: { Accept: 'application/json' },
        });
        if (!res.ok) {
            throw new Error(`Pair failed: ${res.status}`);
        }
        const data = (await res.json()) as PairResponse;
        if (!data.token || data.type !== 'som-transfer') {
            throw new Error('Thiết bị không phải SOM transfer server');
        }

        this.baseUrl = `${parsed.protocol}//${parsed.host}`;
        this.token = data.token;
        return data;
    }

    isPaired(): boolean {
        return !!this.baseUrl && !!this.token;
    }

    getBaseUrl(): string {
        return this.baseUrl;
    }

    async getManifest(): Promise<TransferManifestEntry[]> {
        this.ensurePaired();
        const res = await fetch(`${this.baseUrl}/manifest`, {
            headers: authHeaders(this.token),
        });
        if (!res.ok) {
            throw new Error(`Manifest failed: ${res.status}`);
        }
        const data = await res.json();
        return Array.isArray(data.tracks) ? data.tracks : [];
    }

    async downloadTrack(
        track: TransferManifestEntry,
        onProgress?: (written: number, total: number) => void,
    ): Promise<OfflineTrack> {
        this.ensurePaired();
        await this.ensureDir();

        const localUri = `${DOWNLOAD_DIR}${safeLocalName(track.id)}${extensionFor(track.filename)}`;
        const task = FileSystem.createDownloadResumable(
            `${this.baseUrl}/file/${encodeURIComponent(track.id)}`,
            localUri,
            {
                headers: authHeaders(this.token),
                md5: true,
            },
            (progress) => {
                onProgress?.(
                    progress.totalBytesWritten,
                    progress.totalBytesExpectedToWrite,
                );
            },
        );

        const result = await task.downloadAsync();
        if (!result?.uri) {
            throw new Error('Download failed');
        }

        const info = await FileSystem.getInfoAsync(result.uri, { md5: true });
        if (!info.exists || (track.size > 0 && info.size !== track.size)) {
            await FileSystem.deleteAsync(result.uri, { idempotent: true });
            throw new Error(`Kích thước file không khớp: ${track.title}`);
        }
        if (track.md5 && info.md5 && info.md5.toLowerCase() !== track.md5.toLowerCase()) {
            await FileSystem.deleteAsync(result.uri, { idempotent: true });
            throw new Error(`Checksum không khớp: ${track.title}`);
        }

        const offlineTrack: OfflineTrack = {
            id: track.id,
            title: track.title,
            uploader: track.artist,
            thumbnail: track.thumbnail || '',
            duration: track.duration || 0,
            localUri: result.uri,
            downloadedAt: Date.now(),
        };
        await upsertOfflineTrack(offlineTrack);
        return offlineTrack;
    }

    async uploadTrack(
        track: OfflineTrack,
        onProgress?: (written: number, total: number) => void,
    ): Promise<void> {
        this.ensurePaired();

        const info = await FileSystem.getInfoAsync(track.localUri, { md5: true });
        if (!info.exists || typeof info.size !== 'number') {
            throw new Error(`Không tìm thấy file: ${track.title}`);
        }

        onProgress?.(0, info.size);

        const filename = track.localUri.split('/').pop() || `${safeLocalName(track.id)}.m4a`;
        const res = await FileSystem.uploadAsync(
            `${this.baseUrl}/file/${encodeURIComponent(track.id)}`,
            track.localUri,
            {
                httpMethod: 'PUT',
                uploadType: FileSystem.FileSystemUploadType.BINARY_CONTENT,
                headers: {
                    ...authHeaders(this.token),
                    'Content-Type': 'application/octet-stream',
                    'X-SOM-Filename': filename,
                    'X-SOM-Title': track.title,
                    'X-SOM-Artist': track.uploader,
                    'X-SOM-Duration': String(Math.round(track.duration || 0)),
                    'X-SOM-Thumbnail': track.thumbnail || '',
                    'X-SOM-MD5': info.md5 || '',
                },
            },
        );

        if (res.status < 200 || res.status >= 300) {
            throw new Error(`Upload failed: ${res.status} ${res.body || ''}`.trim());
        }

        onProgress?.(info.size, info.size);
    }

    async syncBothWays(
        onProgress?: (
            side: 'download' | 'upload',
            track: TransferManifestEntry | OfflineTrack,
            written: number,
            total: number,
            index: number,
            count: number,
        ) => void,
    ): Promise<{ downloaded: OfflineTrack[]; uploaded: OfflineTrack[]; skipped: number }> {
        const serverTracks = await this.getManifest();
        const localTracks = await getPlaylist();

        const localById = new Map(localTracks.map((track) => [track.id, track]));
        const serverById = new Map(serverTracks.map((track) => [track.id, track]));

        const toDownload = serverTracks.filter((track) => !localById.has(track.id));
        const toUpload = localTracks.filter((track) => !serverById.has(track.id));

        const downloaded: OfflineTrack[] = [];
        for (let i = 0; i < toDownload.length; i++) {
            const track = toDownload[i];
            const local = await this.downloadTrack(track, (written, total) => {
                onProgress?.('download', track, written, total, i, toDownload.length);
            });
            downloaded.push(local);
        }

        const uploaded: OfflineTrack[] = [];
        for (let i = 0; i < toUpload.length; i++) {
            const track = toUpload[i];
            await this.uploadTrack(track, (written, total) => {
                onProgress?.('upload', track, written, total, i, toUpload.length);
            });
            uploaded.push(track);
        }

        return {
            downloaded,
            uploaded,
            skipped: Math.max(
                0,
                serverTracks.length - toDownload.length,
            ),
        };
    }

    private ensurePaired() {
        if (!this.isPaired()) {
            throw new Error('Chưa kết nối TUI');
        }
    }

    private async ensureDir() {
        const info = await FileSystem.getInfoAsync(DOWNLOAD_DIR);
        if (!info.exists) {
            await FileSystem.makeDirectoryAsync(DOWNLOAD_DIR, { intermediates: true });
        }
    }
}

export default TransferClient;
