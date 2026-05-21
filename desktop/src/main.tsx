import React from 'react';
import ReactDOM from 'react-dom/client';
import { App } from './App';
import { PlayerProvider } from './stores/playerStore';
import './styles.css';

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <PlayerProvider>
      <App />
    </PlayerProvider>
  </React.StrictMode>,
);
