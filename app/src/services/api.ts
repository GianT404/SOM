import * as SecureStore from 'expo-secure-store';
import * as Crypto from 'expo-crypto';

const BASE_URL = process.env.EXPO_PUBLIC_API_URL || 'https://somz.duckdns.org';
const DEVICE_ID_KEY = 'som_device_id';
const TOKEN_KEY = 'som_device_token';

async function getOrCreateDeviceId(): Promise<string> {
    let id = await SecureStore.getItemAsync(DEVICE_ID_KEY);
    if (!id) {
        id = Crypto.randomUUID();
        await SecureStore.setItemAsync(DEVICE_ID_KEY, id);
    }
    return id;
}

async function registerDevice(deviceId: string): Promise<string> {
    const res = await fetch(`${BASE_URL}/api/v1/auth/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ device_id: deviceId }),
    });
    if (!res.ok) {
        throw new Error(`Register failed: ${res.status}`);
    }
    const data = await res.json();
    if (!data.token) {
        throw new Error('Register failed: no token in response');
    }
    await SecureStore.setItemAsync(TOKEN_KEY, data.token);
    return data.token;
}

async function getToken(): Promise<string> {
    let token = await SecureStore.getItemAsync(TOKEN_KEY);
    if (token) return token;
    const deviceId = await getOrCreateDeviceId();
    return registerDevice(deviceId);
}

export interface SearchResult {
    id: string;
    title: string;
    thumbnail: string;
    duration: number;
    uploader: string;
}

export interface LyricLine {
    start: number;
    end: number;
    text: string;
}

export interface LyricsData {
    language: string;
    lines: LyricLine[];
    source?: string;
}

class ApiService {
    private baseUrl: string;
    private retryCount: number = 3;
    private retryDelay: number = 1000; // ms
    private token: string | null = null;
    private tokenPromise: Promise<string> | null = null;

    constructor(baseUrl: string = BASE_URL) {
        this.baseUrl = baseUrl;
    }

    setBaseUrl(url: string) {
        this.baseUrl = url;
    }

    private async ensureToken(): Promise<string> {
        if (this.token) return this.token;
        if (!this.tokenPromise) {
            this.tokenPromise = getToken().then((t) => {
                this.token = t;
                return t;
            });
        }
        return this.tokenPromise;
    }

    private async invalidateToken(): Promise<void> {
        this.token = null;
        this.tokenPromise = null;
        await SecureStore.deleteItemAsync(TOKEN_KEY);
    }

    /**
     * Thêm header X-Device-Token vào request. Khi server trả 401 (token hết
     * hạn do backend restart), xoá token và re-register một lần.
     */
    private async fetchWithFallback(
        url: string,
        options?: RequestInit,
        timeoutMs: number = 10000,
    ): Promise<Response> {
        let lastError: Error | null = null;

        try {
            const token = await this.ensureToken();
            let headers: Record<string, string> = {
                ...((options?.headers as Record<string, string>) || {}),
                'X-Device-Token': token,
            };
            let res = await this.fetchWithTimeout(url, { ...options, headers }, timeoutMs);
            if (res.status === 401) {
                await this.invalidateToken();
                const retryToken = await this.ensureToken();
                headers = { ...headers, 'X-Device-Token': retryToken };
                res = await this.fetchWithTimeout(
                    url,
                    { ...options, headers },
                    timeoutMs,
                );
            }
            if (res.status === 429) {
                await new Promise<void>(resolve => setTimeout(resolve, retryAfterMs(res.headers)));
                res = await this.fetchWithTimeout(url, { ...options, headers }, timeoutMs);
            }
            return res;
        } catch (err) {
            lastError = err instanceof Error ? err : new Error(String(err));
            console.warn(`[API] Request failed:`, lastError.message);
        }

        throw lastError || new Error('Network request failed');
    }

    private async fetchWithTimeout(
        url: string,
        options?: RequestInit,
        timeoutMs: number = 10000,
    ): Promise<Response> {
        const controller = new AbortController();
        const timeout = setTimeout(() => controller.abort(), timeoutMs);
        try {
            return await fetch(url, { ...options, signal: controller.signal });
        } catch (error) {
            if (controller.signal.aborted) {
                throw new Error(`Request timed out after ${timeoutMs}ms`);
            }
            throw error;
        } finally {
            clearTimeout(timeout);
        }
    }

    /**
     * Headers auth cho các API ngoài fetch thường (FileSystem.downloadAsync,
     * createDownloadResumable) — /stream yêu cầu token, redirect CDN không cần.
     */
    async getStreamHeaders(): Promise<Record<string, string>> {
        return { 'X-Device-Token': await this.ensureToken() };
    }

    async search(query: string): Promise<SearchResult[]> {
        const url = `${this.baseUrl}/api/v1/search?q=${encodeURIComponent(query)}`;
        const res = await this.fetchWithFallback(url);
        if (!res.ok) throw new Error(`Search failed: ${res.status}`);
        return res.json();
    }

    /**
     * URL stream audio. clean=true → backend chạy ffmpeg silenceremove
     * (cắt khoảng lặng < -50dB kéo dài >= 1s) — chậm hơn vì transcode realtime.
     */
    getStreamUrl(videoId: string, clean = false): string {
        const url = `${this.baseUrl}/api/v1/stream?id=${encodeURIComponent(videoId)}`;
        return clean ? `${url}&clean=1` : url;
    }

    async resolveUrl(videoId: string): Promise<{ url: string; title: string; safeName: string }> {
        const url = `${this.baseUrl}/api/v1/resolve?id=${encodeURIComponent(videoId)}`;
        const res = await this.fetchWithFallback(url, undefined, 55000);
        if (!res.ok) throw new Error(`Resolve failed: ${res.status}`);
        return res.json();
    }

    async getLyrics(
        videoId: string,
        meta?: { title?: string; artist?: string; duration?: number },
        source?: 'lrclib' | 'youtube'
    ): Promise<LyricsData[]> {
        const params = new URLSearchParams({ id: videoId });
        if (meta?.title) params.set('title', meta.title);
        if (meta?.artist) params.set('artist', meta.artist);
        if (meta?.duration) params.set('duration', String(Math.round(meta.duration)));
        if (source) params.set('source', source);

        const url = `${this.baseUrl}/api/v1/lyrics?${params.toString()}`;
        const res = await this.fetchWithFallback(url);
        if (!res.ok) {
            let errorMessage = `Lyrics failed: ${res.status}`;
            try {
                const errData = await res.json();
                if (errData.error) errorMessage = errData.error;
            } catch (e) { /* ignore */ }
            throw new Error(errorMessage);
        }
        return res.json();
    }

    async healthCheck(): Promise<boolean> {
        try {
            const res = await this.fetchWithTimeout(`${this.baseUrl}/health`);
            return res.ok;
        } catch {
            return false;
        }
    }

    /**
     * Get current backend URL (local or cloud)
     */
    getCurrentUrl(): string {
        return this.baseUrl;
    }

    /**
     * Check if using local backend
     */
    // isLocalBackend(): boolean {
    //     return this.baseUrl === LOCALHOST_URL;
    // }
}

function retryAfterMs(headers: Headers): number {
    const raw = headers.get('Retry-After');
    const seconds = Number(raw);
    if (Number.isFinite(seconds) && seconds > 0) {
        return Math.min(seconds, 30) * 1000;
    }

    const retryAt = raw ? Date.parse(raw) : NaN;
    if (Number.isFinite(retryAt)) {
        return Math.min(Math.max(retryAt - Date.now(), 1000), 30000);
    }
    return 1000;
}

export const api = new ApiService();
export default api;
