import { invoke } from '@tauri-apps/api/core';
import { CheckCircle2, FolderDown, Server, XCircle } from 'lucide-react';
import { useState } from 'react';

export function SettingsPage({ backendReady, backendUrl }: { backendReady: boolean; backendUrl: string }) {
  const [downloadDir, setDownloadDir] = useState('');

  return (
    <section className="page settings-page">
      <header className="topbar">
        <div>
          <p className="eyebrow">Desktop</p>
          <h1>Settings</h1>
        </div>
      </header>
      <div className="settings-grid">
        <article className="settings-card neo-card shadow-sm">
          <Server size={24} />
          <div>
            <h2>Backend sidecar</h2>
            <p>{backendUrl}</p>
          </div>
          {backendReady ? <CheckCircle2 className="ok" /> : <XCircle className="bad" />}
        </article>
        <article className="settings-card neo-card shadow-sm">
          <FolderDown size={24} />
          <div>
            <h2>Offline downloads</h2>
            <p>{downloadDir || 'Stored in the Tauri app data directory.'}</p>
          </div>
          <button
            className="small-button"
            onClick={() => {
              invoke<string>('reveal_downloads_dir').then(setDownloadDir).catch((err) => setDownloadDir(String(err)));
            }}
          >
            Show path
          </button>
        </article>
      </div>
    </section>
  );
}

