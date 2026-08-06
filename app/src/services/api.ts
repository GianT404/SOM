import * as SecureStore from 'expo-secure-store';
import * as Crypto from 'expo-crypto';

// Set via app/.env (gitignored): EXPO_PUBLIC_API_URL=https://somz.duckdns.org
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
    private async fetchWithFallback(url: string, options?: RequestInit): Promise<Response> {
        let lastError: Error | null = null;

        try {
            const token = await this.ensureToken();
            const headers: Record<string, string> = {
                ...((options?.headers as Record<string, string>) || {}),
                'X-Device-Token': token,
            };
            const res = await this.fetchWithTimeout(url, { ...options, headers });
            if (res.status === 401) {
                await this.invalidateToken();
                const retryToken = await this.ensureToken();
                const retryRes = await this.fetchWithTimeout(url, {
                    ...options,
                    headers: { ...headers, 'X-Device-Token': retryToken },
                });
                return retryRes;
            }
            return res;
        } catch (err) {
            lastError = err instanceof Error ? err : new Error(String(err));
            console.warn(`[API] Request failed:`, lastError.message);
        }

        throw lastError || new Error('Network request failed');
    }

    private async fetchWithTimeout(url: string, options?: RequestInit): Promise<Response> {
        return Promise.race([
            fetch(url, options),
            new Promise<Response>((_, reject) =>
                setTimeout(() => reject(new Error('Timeout')), 10000)
            ),
        ]);
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

    getStreamUrl(videoId: string): string {
        return `${this.baseUrl}/api/v1/stream?id=${encodeURIComponent(videoId)}`;
    }

    async resolveUrl(videoId: string): Promise<{ url: string; title: string; safeName: string }> {
        const url = `${this.baseUrl}/api/v1/resolve?id=${encodeURIComponent(videoId)}`;
        const res = await this.fetchWithFallback(url);
        if (!res.ok) throw new Error(`Resolve failed: ${res.status}`);
        return res.json();
    }

    async getLyrics(
        videoId: string,
        meta?: { title?: string; artist?: string; duration?: number }
    ): Promise<LyricsData[]> {
        const params = new URLSearchParams({ id: videoId });
        if (meta?.title) params.set('title', meta.title);
        if (meta?.artist) params.set('artist', meta.artist);
        if (meta?.duration) params.set('duration', String(Math.round(meta.duration)));

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

export const api = new ApiService();
export default api;
