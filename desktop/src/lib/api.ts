import { invoke } from '@tauri-apps/api/core';
import type { LyricsData, SearchResult } from './types';

const FALLBACK_BASE_URL = 'http://127.0.0.1:8080';
const HEALTH_TIMEOUT_MS = 5000;
const DEFAULT_TIMEOUT_MS = 30000;
const SEARCH_TIMEOUT_MS = 120000;

class ApiClient {
  private baseUrl = FALLBACK_BASE_URL;

  async init() {
    try {
      this.baseUrl = await invoke<string>('get_backend_url');
    } catch {
      this.baseUrl = FALLBACK_BASE_URL;
    }
  }

  setBaseUrl(baseUrl: string) {
    this.baseUrl = baseUrl.replace(/\/$/, '');
  }

  getBaseUrl() {
    return this.baseUrl;
  }

  async healthCheck() {
    try {
      const response = await fetchWithTimeout(`${this.baseUrl}/health`, HEALTH_TIMEOUT_MS);
      return response.ok;
    } catch {
      return false;
    }
  }

  async search(query: string): Promise<SearchResult[]> {
    const response = await this.fetchJson(
      `${this.baseUrl}/api/v1/search?q=${encodeURIComponent(query)}`,
      SEARCH_TIMEOUT_MS,
      'Search timed out while waiting for yt-dlp. Try again in a moment.',
    );
    return response as SearchResult[];
  }

  getStreamUrl(videoId: string) {
    return `${this.baseUrl}/api/v1/stream?id=${encodeURIComponent(videoId)}`;
  }

  async getLyrics(
    videoId: string,
    meta?: { title?: string; artist?: string; duration?: number },
  ): Promise<LyricsData[]> {
    const params = new URLSearchParams({ id: videoId });
    if (meta?.title) params.set('title', meta.title);
    if (meta?.artist) params.set('artist', meta.artist);
    if (meta?.duration) params.set('duration', String(Math.round(meta.duration)));
    return (await this.fetchJson(`${this.baseUrl}/api/v1/lyrics?${params.toString()}`)) as LyricsData[];
  }

  private async fetchJson(url: string, timeoutMs = DEFAULT_TIMEOUT_MS, timeoutMessage = 'Request timed out') {
    let response: Response;
    try {
      response = await fetchWithTimeout(url, timeoutMs);
    } catch (err) {
      if (isAbortedFetch(err)) {
        throw new Error(timeoutMessage);
      }
      throw err;
    }

    if (!response.ok) {
      let detail = response.statusText;
      try {
        const body = await response.json();
        detail = body.error || detail;
      } catch {
        // Keep status text.
      }
      throw new Error(detail || `HTTP ${response.status}`);
    }
    return response.json();
  }
}

export const api = new ApiClient();

function fetchWithTimeout(url: string, timeoutMs: number) {
  const controller = new AbortController();
  const timeoutId = window.setTimeout(() => controller.abort(), timeoutMs);

  return fetch(url, { signal: controller.signal }).finally(() => {
    window.clearTimeout(timeoutId);
  });
}

function isAbortedFetch(err: unknown) {
  if (err instanceof DOMException && (err.name === 'AbortError' || err.name === 'TimeoutError')) {
    return true;
  }

  if (err instanceof Error) {
    const message = err.message.toLowerCase();
    return message.includes('abort') || message.includes('timed out') || message.includes('timeout');
  }

  return false;
}
