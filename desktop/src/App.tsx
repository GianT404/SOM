import { useEffect, useMemo, useState } from 'react';
import { AppShell } from './components/AppShell';
import { HomePage } from './components/HomePage';
import { LibraryPage } from './components/LibraryPage';
import { LyricsPanel } from './components/LyricsPanel';
import { SearchPage } from './components/SearchPage';
import { SettingsPage } from './components/SettingsPage';
import { api } from './lib/api';
import type { ViewKey } from './lib/types';

export function App() {
  const [view, setView] = useState<ViewKey>('home');
  const [backendReady, setBackendReady] = useState(false);
  const [backendUrl, setBackendUrl] = useState(api.getBaseUrl());
  const [backendChecked, setBackendChecked] = useState(false);

  useEffect(() => {
    let active = true;

    const initBackend = async () => {
      await api.init();
      setBackendUrl(api.getBaseUrl());

      for (let attempt = 0; attempt < 15; attempt += 1) {
        const ready = await api.healthCheck();
        if (!active) return;
        setBackendReady(ready);
        setBackendChecked(true);
        setBackendUrl(api.getBaseUrl());
        if (ready) return;
        await new Promise((resolve) => window.setTimeout(resolve, 700));
      }
    };

    initBackend().catch(() => {
      if (!active) return;
      setBackendReady(false);
      setBackendChecked(true);
      setBackendUrl(api.getBaseUrl());
    });

    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      const isTyping = target?.tagName === 'INPUT' || target?.tagName === 'TEXTAREA';
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'f') {
        event.preventDefault();
        setView('search');
        window.setTimeout(() => document.getElementById('desktop-search-input')?.focus(), 50);
      }
      if (!isTyping && event.key === 'ArrowLeft') window.dispatchEvent(new Event('som-previous'));
      if (!isTyping && event.key === 'ArrowRight') window.dispatchEvent(new Event('som-next'));
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  const content = useMemo(() => {
    if (view === 'search') {
      return <SearchPage backendReady={backendReady} backendChecked={backendChecked} onNavigate={setView} />;
    }
    if (view === 'library') return <LibraryPage />;
    if (view === 'lyrics') return <LyricsPanel standalone />;
    if (view === 'settings') return <SettingsPage backendReady={backendReady} backendUrl={backendUrl} />;
    return <HomePage onNavigate={setView} />;
  }, [backendChecked, backendReady, backendUrl, view]);

  return (
    <AppShell
      backendReady={backendReady}
      backendUrl={backendUrl}
      activeView={view}
      onNavigate={setView}
    >
      {content}
    </AppShell>
  );
}
