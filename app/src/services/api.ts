const BASE_URL = 'http://192.168.1.5:8080';

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

    constructor(baseUrl: string = BASE_URL) {
        this.baseUrl = baseUrl;
    }

    setBaseUrl(url: string) {
        this.baseUrl = url;
    }

    async search(query: string): Promise<SearchResult[]> {
        const res = await fetch(`${this.baseUrl}/api/v1/search?q=${encodeURIComponent(query)}`);
        if (!res.ok) throw new Error(`Search failed: ${res.status}`);
        return res.json();
    }

    getStreamUrl(videoId: string): string {
        return `${this.baseUrl}/api/v1/stream?id=${encodeURIComponent(videoId)}`;
    }

    async getLyrics(videoId: string): Promise<LyricsData[]> {
        const res = await fetch(`${this.baseUrl}/api/v1/lyrics?id=${encodeURIComponent(videoId)}`);
        if (!res.ok) {
            let errorMessage = `Lyrics failed: ${res.status}`;
            try {
                const errData = await res.json();
                if (errData.error) errorMessage = errData.error;
            } catch (e) {
                // ignore JSON parse error
            }
            throw new Error(errorMessage);
        }
        return res.json();
    }

    async healthCheck(): Promise<boolean> {
        try {
            const res = await fetch(`${this.baseUrl}/health`);
            return res.ok;
        } catch {
            return false;
        }
    }
}

export const api = new ApiService();
export default api;
