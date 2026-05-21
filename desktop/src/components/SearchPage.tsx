import { Loader2, Music2, Search, X } from 'lucide-react';
import { useCallback, useState } from 'react';
import { api } from '../lib/api';
import type { SearchResult, ViewKey } from '../lib/types';
import { usePlayer } from '../stores/playerStore';
import { TrackTable } from './TrackTable';

export function SearchPage({
  backendReady,
  backendChecked,
  onNavigate,
}: {
  backendReady: boolean;
  backendChecked: boolean;
  onNavigate: (view: ViewKey) => void;
}) {
  const { play } = usePlayer();
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [searched, setSearched] = useState(false);

  const runSearch = useCallback(async (term = query) => {
    const next = term.trim();
    if (!next || !backendReady) return;
    setQuery(next);
    setLoading(true);
    setSearched(true);
    setError('');
    try {
      setResults(await api.search(next));
    } catch (err) {
      setResults([]);
      setError(err instanceof Error ? err.message : 'Search failed');
    } finally {
      setLoading(false);
    }
  }, [backendReady, query]);

  return (
    <section className="page search-page">
      <form
        className="search-box"
        onSubmit={(event) => {
          event.preventDefault();
          void runSearch();
        }}
      >
        <input
          id="desktop-search-input"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Muốn tìm gì?"
        />
        <button type="button" onClick={() => (query ? (setQuery(''), setResults([]), setSearched(false)) : void runSearch())}>
          {query ? <X size={20} /> : <Search size={20} />}
        </button>
      </form>

      {!backendReady && (
        <div className="empty-state error">
          <Music2 size={42} />
          <h2>{backendChecked ? 'Backend offline' : 'Starting backend'}</h2>
          <p>{backendChecked ? 'Local API is not reachable yet.' : 'Waiting for the local Go sidecar.'}</p>
        </div>
      )}
      {loading && (
        <div className="empty-state">
          <Loader2 className="spin" size={42} />
          <p>Đang tìm...</p>
        </div>
      )}
      {backendReady && !loading && error && (
        <div className="empty-state error">
          <Music2 size={42} />
          <h2>Search failed</h2>
          <p>{error}</p>
        </div>
      )}
      {backendReady && !loading && !error && searched && results.length === 0 && (
        <div className="empty-state">
          <Music2 size={42} />
          <h2>No results found</h2>
        </div>
      )}
      {backendReady && !loading && results.length > 0 && (
        <TrackTable
          tracks={results}
          sourceLabel={() => 'YouTube'}
          onPlay={(track) => {
            void play(track, results);
            onNavigate('lyrics');
          }}
        />
      )}
    </section>
  );
}
