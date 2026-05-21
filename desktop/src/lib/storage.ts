import type { OfflineTrack } from './types';

const PLAYLIST_KEY = 'som_desktop_playlist';
const DELETED_KEY = 'som_desktop_deleted_playlist';

function readList(key: string): OfflineTrack[] {
  try {
    return JSON.parse(localStorage.getItem(key) || '[]') as OfflineTrack[];
  } catch {
    return [];
  }
}

function writeList(key: string, tracks: OfflineTrack[]) {
  localStorage.setItem(key, JSON.stringify(tracks));
}

export function getPlaylist() {
  return readList(PLAYLIST_KEY);
}

export function savePlaylist(tracks: OfflineTrack[]) {
  writeList(PLAYLIST_KEY, tracks);
  window.dispatchEvent(new Event('som-playlist-changed'));
}

export function getDeletedPlaylist() {
  return readList(DELETED_KEY);
}

export function saveDeletedPlaylist(tracks: OfflineTrack[]) {
  writeList(DELETED_KEY, tracks);
  window.dispatchEvent(new Event('som-playlist-changed'));
}

export function softDeleteTrack(id: string) {
  const playlist = getPlaylist();
  const track = playlist.find((item) => item.id === id);
  if (!track) return;
  savePlaylist(playlist.filter((item) => item.id !== id));
  saveDeletedPlaylist([...getDeletedPlaylist(), { ...track, deletedAt: Date.now() }]);
}

export function restoreTrack(id: string) {
  const deleted = getDeletedPlaylist();
  const track = deleted.find((item) => item.id === id);
  if (!track) return;
  saveDeletedPlaylist(deleted.filter((item) => item.id !== id));
  savePlaylist([...getPlaylist(), { ...track, deletedAt: undefined }]);
}

export function permanentlyDeleteTrack(id: string) {
  saveDeletedPlaylist(getDeletedPlaylist().filter((item) => item.id !== id));
}

export function upsertPlaylistTrack(track: OfflineTrack) {
  const playlist = getPlaylist();
  const next = playlist.some((item) => item.id === track.id)
    ? playlist.map((item) => (item.id === track.id ? track : item))
    : [...playlist, track];
  savePlaylist(next);
}

