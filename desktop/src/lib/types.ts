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

export interface Track extends SearchResult {
  localPath?: string;
  lyrics?: LyricsData[];
}

export interface OfflineTrack extends Track {
  localPath: string;
  downloadedAt: number;
  deletedAt?: number;
}

export type ViewKey = 'home' | 'search' | 'library' | 'lyrics' | 'settings';

