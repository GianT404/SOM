// const BASE_URL = 'http://localhost:8080';
const BASE_URL = 'http://192.168.2.29:8080';

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

    constructor(baseUrl: string = BASE_URL) {
        this.baseUrl = baseUrl;
    }

    setBaseUrl(url: string) {
        this.baseUrl = url;
    }

    /**
     * Thử URL chính trước, fallback sang cloud nếu thất bại
     */
    private async fetchWithFallback(url: string, options?: RequestInit): Promise<Response> {
        let lastError: Error | null = null;

        // Try primary URL (localhost)
        try {
            console.log(`[API] Fetching from primary: ${url}`);
            const res = await Promise.race([
                fetch(url, options),
                new Promise<Response>((_, reject) =>
                    setTimeout(() => reject(new Error('Timeout')), 10000)
                ),
            ]);
            if (res.ok) return res;
            lastError = new Error(`${res.status}: ${res.statusText}`);
        } catch (err) {
            lastError = err instanceof Error ? err : new Error(String(err));
            console.warn(`[API] Primary failed:`, lastError.message);
        }

        // Fallback to cloud URL
        // if (this.baseUrl !== this.cloudUrl) {
        //     try {
        //         const cloudUrlStr = url.replace(this.baseUrl, this.cloudUrl);
        //         console.log(`[API] Falling back to cloud: ${cloudUrlStr}`);
        //         const res = await Promise.race([
        //             fetch(cloudUrlStr, options),
        //             new Promise<Response>((_, reject) =>
        //                 setTimeout(() => reject(new Error('Timeout')), 10000)
        //             ),
        //         ]);
        //         if (res.ok) return res;
        //         lastError = new Error(`Cloud: ${res.status}`);
        //     } catch (err) {
        //         lastError = err instanceof Error ? err : new Error(String(err));
        //         console.warn(`[API] Cloud fallback failed:`, lastError.message);
        //     }
        // }

        throw lastError || new Error('Network request failed');
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
            const res = await Promise.race([
                fetch(`${this.baseUrl}/health`),
                new Promise<Response>((_, reject) =>
                    setTimeout(() => reject(new Error('Timeout')), 5000)
                ),
            ]);
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
