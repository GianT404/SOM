import { invoke } from '@tauri-apps/api/core';
import { CheckCircle2, FolderDown, Server, XCircle } from 'lucide-react';
import { useState } from 'react';

export function SettingsPage({ backendReady, backendUrl }: { backendReady: boolean; backendUrl: string }) {
  const [downloadDir, setDownloadDir] = useState('');

  return (
    <section className="page settings-page">
      <div className="settings-grid">
        <article className="settings-card">
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
