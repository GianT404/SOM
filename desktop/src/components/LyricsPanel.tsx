import { Loader2, Subtitles } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { api } from '../lib/api';
import type { LyricLine, LyricsData } from '../lib/types';
import { usePlayer } from '../stores/playerStore';

export function LyricsPanel({ compact = false, standalone = false }: { compact?: boolean; standalone?: boolean }) {
  const { currentTrack, position } = usePlayer();
  const [lyricsData, setLyricsData] = useState<LyricsData[]>([]);
  const [language, setLanguage] = useState('vi');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const cache = useRef<Record<string, LyricsData[]>>({});
  const currentSec = position / 1000;

  useEffect(() => {
    if (!currentTrack) return;
    if (currentTrack.lyrics?.length) {
      setLyricsData(currentTrack.lyrics);
      setError('');
      return;
    }
    if (cache.current[currentTrack.id]) {
      setLyricsData(cache.current[currentTrack.id]);
      setError('');
      return;
    }
    let active = true;
    setLoading(true);
    setError('');
    api.getLyrics(currentTrack.id, {
      title: currentTrack.title,
      artist: currentTrack.uploader,
      duration: currentTrack.duration,
    })
      .then((data) => {
        if (!active) return;
        cache.current[currentTrack.id] = data;
        setLyricsData(data);
      })
      .catch((err) => {
        if (!active) return;
        setLyricsData([]);
        setError(err instanceof Error ? err.message : 'No lyrics available');
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [currentTrack]);

  const activeLyrics = useMemo<LyricLine[]>(() => {
    const selected = lyricsData.find((item) => item.language === language) ?? lyricsData[0];
    return selected?.lines ?? [];
  }, [language, lyricsData]);

  const activeIndex = activeLyrics.findIndex((line, index) => {
    const next = activeLyrics[index + 1];
    return currentSec >= line.start && (!next || currentSec < next.start);
  });

  if (!currentTrack) {
    return (
      <section className={`lyrics-panel ${compact ? 'compact' : ''} ${standalone ? 'standalone' : ''}`}>
        <div className="empty-state mini">
          <Subtitles size={40} />
          <p>No track playing</p>
        </div>
      </section>
    );
  }

  return (
    <section className={`lyrics-panel ${compact ? 'compact' : ''} ${standalone ? 'standalone' : ''}`}>
      <header className="lyrics-header">
        <img src={currentTrack.thumbnail} alt="" />
        <div>
          <h2>{standalone ? 'Lyrics' : currentTrack.title}</h2>
          <p>{currentTrack.uploader}</p>
        </div>
      </header>

      {lyricsData.length > 1 && (
        <div className="language-tabs">
          {lyricsData.map((item) => (
            <button
              key={item.language}
              className={language === item.language ? 'active' : ''}
              onClick={() => setLanguage(item.language)}
            >
              {item.language.toUpperCase()}
            </button>
          ))}
        </div>
      )}

      {loading ? (
        <div className="empty-state mini">
          <Loader2 className="spin" size={36} />
          <p>Loading lyrics</p>
        </div>
      ) : error ? (
        <div className="empty-state mini">
          <Subtitles size={38} />
          <p>{error}</p>
        </div>
      ) : activeLyrics.length ? (
        <div className="lyrics-scroll">
          {activeLyrics.map((line, index) => (
            <p key={`${line.start}-${index}`} className={index === activeIndex ? 'active' : ''}>
              {line.text}
            </p>
          ))}
        </div>
      ) : (
        <div className="empty-state mini">
          <Subtitles size={38} />
          <p>No lyrics available</p>
        </div>
      )}
    </section>
  );
}

