<p align="center">
  <img src="./assets/logo.svg" alt="SOM Logo" width="120" />
</p>

<h1 align="center">SOM</h1>

<p align="center">
  <b>Stream and play local music </b>
  <br>
  <b>Supports Linux, Windows, Android, and iOS.</b>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/React_Native-0.83-61DAFB?logo=react&logoColor=white" />
  <img src="https://img.shields.io/badge/Expo-55-000020?logo=expo&logoColor=white" />
  <img src="https://img.shields.io/badge/TypeScript-5.3-3178C6?logo=typescript&logoColor=white" />
</p>

<p align="center">
  <img src="./assets/thumbnail.png" alt="SOM Logo" width="100%" />
</p>

## Demo
https://github.com/user-attachments/assets/b2050ea6-6621-4a2c-a366-9326a810218e

## Audio Visualizer
https://github.com/user-attachments/assets/d7bf017b-7a73-4f7e-8d07-964e5f460249

---

## Features

| Feature | Description | 
|---------|-------------|
|  **Search** | Search YouTube for any song or artist |
|  **Stream** | Stream audio directly without downloading the full video |
|  **Offline Playback** | Download tracks (`.opus`/`.mp3`/`.mp4`) for offline listening |
|  **Playlists** | Create, manage, and play custom playlists (TUI) |
|  **Synced Lyrics** | Real-time synced lyrics via LRCLib + YouTube subtitles fallback, with language selection and millisecond-precise pre-roll seeking |
|  **Audio Presets** | Quick audio filtering via command menu: Normal, Bass Boost, Nightcore, Daycore, and Lo-Fi |
|  **Media Controls** | Lock screen & notification controls (play, pause, skip, seek) |
|  **Shuffle** | 	Randomized playback that avoids replaying recently-played tracks and back-to-back songs by the same artist |
|  **Audio Visualizer** | Live 2D/3D visualizer driven by real-time system audio capture (TUI) |
|  **Dynamic Theming** | Album art-based color extraction for immersive UI |
|  **Audio Settings** | Configurable buffer size and sample rate |
|  **CLI & Completion** | Powered by Cobra with built-in auto-completion scripts for Zsh, Bash, and Fish (`som completion [shell]`) |
|  **Self Install/Update** | `som --install`, `som --upgrade`, `--check-update`, and `--uninstall` for one-command setup and updates |
|  **Resilient YouTube Access** | Automatic yt-dlp client fallback against YouTube/CDN 403s, optional cookies & extra args via env vars

---

## Architecture

```
SOM/
├── cmd/
│   ├── server/          # Go backend entry point
│   │   └── main.go      # HTTP server (chi router)
│   └── som/             # TUI (terminal) entry point (Cobra CLI)
│       ├── main.go      # Root command & flags setup
│       ├── install.go   # Self-installation routines
│       └── update.go    # Self-update & GitHub release checker
├── internal/
│   ├── handler/         # API route handlers
│   │   ├── search.go    # GET /api/v1/search
│   │   ├── stream.go    # GET /api/v1/stream
│   │   ├── lyrics.go    # GET /api/v1/lyrics
│   │   └── resolve.go   # GET /api/v1/resolve
│   ├── backend/         # Device auth + rate limiting (used by cmd/server)
│   ├── cache/           # In-memory TTL cache
│   ├── domain/          # Shared models & MusicProvider interface
│   ├── local/           # DirectProvider — in-process backend for the TUI
│   ├── scraper/         # YouTube data extraction
│   │   ├── ytdlp.go     # yt-dlp wrapper
│   │   ├── lrclib.go    # LRCLib lyrics API
│   │   └── ...          # VTT parser, fallback scrapers
│   ├── storage/         # SQLite persistence (WAL, FTS5, pure Go)
│   │   ├── db.go        # Schema, migrations, DB connection
│   │   ├── playlists.go # Playlist CRUD
│   │   ├── downloads.go # Local file management, FTS search, filesystem import
│   │   └── history.go   # Listen history
│   ├── tui/
│   │   ├── api/               # HTTP client for remote mode (--server)
│   │   ├── audio/             # System audio capture + FFT for the visualizer
│   │   ├── player/            # ffmpeg-based decode + oto playback (opus_player.go)
│   │   ├── bindeps/           # Bundled binary dependency helpers
│   │   ├── logbuf/            # Thread-safe in-memory ring-buffer logger (crash dump support)
│   │   ├── cmd/tui/           # Standalone TUI entry point
│   │   └── ui/                # TUI components (Bubble Tea)
│   │       ├── app.go           # Main app loop, sidebar, progress bar, key handling
│   │       ├── bocchi_frame.go  # (My wifu XD)
│   │       ├── browse.go        # Left panel (search/downloads/local)
│   │       ├── nowplaying.go    # Right panel (lyrics, track info)
│   │       ├── search.go        # Search input view
│   │       ├── downloads.go     # Local file scanning
│   │       ├── playlists.go     # Playlist create/add/play/delete UI
│   │       ├── palette.go       # Live 2D audio visualizer (opened with \)
│   │       ├── visualizer_3d.go # 3D wireframe visualizer mode
│   │       ├── help.go          # Keyboard shortcuts popup (?)
│   │       ├── sidebar.go       # Sidebar definition
│   │       ├── logs.go          # Ring-buffer logger
│   │       ├── styles.go        # Styles & nerd-font icons
│   │       ├── commands.go      # Bubble Tea commands
│   │       ├── box.go           # Box drawing helper
│   │       ├── msgs.go          # Message types
│   │       ├── cache.go         # Lyrics/search caching
│   │       └── lyrics.go        # Lyric line parsing
│   └── cleaner/          # Audio stream processing
├── app/                 # React Native (Expo) mobile app
│   ├── src/
│   │   ├── screens/     # HomeScreen, SearchScreen, NowPlayingScreen, ...
│   │   ├── components/  # MiniPlayer, MusicCard, SyncedLyrics, ...
│   │   ├── contexts/    # PlayerContext (global audio state)
│   │   ├── services/    # API client, media controls, playlist store
│   │   ├── hooks/       # Custom React hooks
│   │   ├── navigation/  # React Navigation setup
│   │   └── theme/       # Colors, typography
│   └── assets/          # App icons, splash screen
├── desktop/             # Tauri v2 desktop app (React + Vite + TypeScript)
├── docs/
│   ├── index.html       # Inline API documentation
│   └── openapi.yaml     # OpenAPI spec
└── go.mod               # Go module definition
```

**Backend** — A lightweight Go HTTP server that proxies YouTube audio and lyrics (can run embedded inside the TUI).

**Terminal UI** — A Bubble Tea TUI that embeds the backend in-process, featuring keyboard-driven music search, stream, download, lyrics, and local file playback.

**Mobile Frontend** — A React Native app built with Expo, featuring background audio playback, offline downloads, and synced lyrics.

---

## Getting Started

### Prerequisites

- **Go** 1.26+
- **Node.js** 18+ & **npm**
- **yt-dlp** installed and available in `$PATH`
- **Android Studio** (for Android builds) or **Xcode** (for iOS builds)
- **Expo CLI** — `npm install -g expo-cli`

### 1. Clone the repository

```bash
git clone https://github.com/GianT404/SOM.git
cd SOM
```

### 2. Start the backend

```bash
cd cmd/server
go build -o server .
./server
```

The server starts on port `8080` by default. Configure with environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `HOST` | (all interfaces) | Bind host |
| `YTDLP_PATH` | `yt-dlp` | Path to yt-dlp binary |
| `SOM_DOWNLOAD_DIR` | `~/.local/share/som` | Directory to store downloaded tracks (overridden by `--download-dir` flag) |
| `SOM_API_KEY` | — | Require an API key for requests |
| `BANNED_DEVICES` | — | Comma-separated device IDs to ban |
| `TRUSTED_PROXIES` | — | Comma-separated IP/CIDR allowed to supply `X-Forwarded-For` |
| `SOM_YTDLP_CLIENTS` | `web_embedded,web,tv_embedded,android,mweb,ios` | Comma-separated yt-dlp player clients to try in order (used by both server and TUI) |
| `SOM_YTDLP_COOKIES` | — | Path to a Netscape `cookies.txt` from a logged-in browser session (`--cookies`) |
| `SOM_YTDLP_ARGS` | — | Extra yt-dlp args, appended verbatim to every yt-dlp invocation |
| `SOM_YTDLP_FASTPATH` | (server on, TUI off) | Set `1` to resolve stream URLs with the in-process `youtube/v2` lib instead of spawning yt-dlp (server enables this automatically) |
| `YTDLP_CACHE_DIR` | — | Directory for the yt-dlp HTTP cache |

### 3. Start the mobile app

```bash
cd app
npm install --legacy-peer-deps
npx expo start
```

Then press `a` to open on Android emulator, or scan the QR code with **Expo Go**.

### Build APK (Android)

```bash
cd app
npx expo run:android
```

### TUI (Terminal UI)

A standalone terminal music player — stream, download, and listen to music directly from the command line. No desktop environment required.

**Build & run:**

```bash
cd cmd/som
go build -o som .
./som
```

**Sidebar tabs (`1`-`6` or `Tab`):** Search, Downloads, Playlists, Lyrics, Logs.

**Controls:**

| Key | Action |
|-----|--------|
| `?` | Toggle help popup (full shortcut list) |
| `Tab` | Cycle through sidebar tabs |
| `1`-`6` | Jump directly to a tab |
| `/` | Focus search input (Playlists tab: create a new playlist) |
| `:` | Command popup — Rename title, Delete track, Move to playlist, Show file info |
| `\` | Open live audio visualizer |
| `Enter` | Play selected track |
| `Space` | Play / pause |
| `]` / `[` | Next / previous track |
| `r` | Toggle shuffle (random) |
| `←` / `→` | Seek -/+ 5s |
| `+` / `-` | Volume up / down |
| `Delete` | Remove selected playlist / track |
| `d` | Download selected track |
| `l` | Choose lyrics language |
| `pgup` / `pgdown` | scroll lyric page|
| `alt + q` | Quit from anywhere (also during startup) |

**Features:**

- YouTube search and stream via embedded backend (Go server runs in-process, no separate process needed)
- Download tracks for offline playback in `.opus`/`.mp3`/`.mp4` format
- Playlist management — create, add tracks, play, and delete playlists
- Command popup (`:`) — rename a track's title, delete a track, move a track to another playlist, show file info (size/duration/bitrate), and remove from the current playlist
- LRCLib synced lyrics with multi-language selection and auto-fallback to YouTube subtitles
- Local `.opus`/`.mp3`/`.mp4` file scanning with `ffprobe` duration detection, persisted in SQLite with FTS5 full-text search
- Live audio visualizer (`\`) with 2D bar mode and a 3D wireframe mode (toggle with `l` while open), driven by real-time system audio capture
- Progress bar with control buttons (prev / play-pause / next / shuffle)
- Resilient YouTube streaming via client fallback chain (see *YouTube Stability & Cookies*)
- Built-in self-installer and self-updater (see below)

**Logging (`6` / Logs tab):**

- Captures **all** `log` output in-process (TUI + backend) into a thread-safe in-memory ring buffer (last 2000 lines) — search, stream, download, and lyrics actions are logged on the happy path in local mode.
- Logs live **only in RAM** — quitting normally (`q`/`Ctrl+C`) never touches the disk; the old `~/som_debug.log` file was removed.
- On a **crash (panic)** or **SIGQUIT**, the ring buffer is dumped synchronously to `~/crash_som_tui_<timestamp>.log` for debugging.

**Cross-platform builds:**

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o som-linux-amd64 ./cmd/som
```

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o som-windows-amd64.exe ./cmd/som
```

**CLI flags:**

| Flag | Description |
|------|-------------|
| `--server <URL>` | Run in remote mode, pointing to a SOM backend instead of the in-process one |
| `--download-dir <DIR>` | Directory to store downloaded tracks (default: `~/.local/share/som`) |
| `--install` | Copy this binary to `/usr/local/bin` (or the Windows equivalent) so `som` runs from anywhere |
| `--upgrade` | Download and install the latest SOM release from GitHub |
| `--check-update` | Check whether a newer SOM release exists without installing |
| `--uninstall` | Remove the installed `som` binary |
| `--update-ytdlp` | Update the bundled yt-dlp binary to the latest version |
| `--version` | Print the current version and exit |
| `--changelog` | Print the commits of the current version |
| `--help`| Help for som |

> Requires `yt-dlp` and `ffmpeg` in `PATH` (audio is decoded via `ffmpeg` and played back with `oto`. `ffprobe` is also used for local file duration detection.

---
## Installation

You can automatically install SOM and its dependencies (yt-dlp, ffmpeg) using the following commands:

```bash
#Linux
curl -fsSL https://raw.githubusercontent.com/GianT404/SOM/main/scripts/install.sh | bash
```

```bash
#Windows (PowerShell)
curl.exe -O https://raw.githubusercontent.com/GianT404/SOM/main/scripts/install.ps1 | iex
```
## Shell Auto-Completion Setup
Since SOM uses Cobra, you can seamlessly enable auto-completion for your shell of choice:

```bash
#zsh
mkdir -p ~/.zfunc
som completion zsh > ~/.zfunc/_som
```
>(Ensure fpath=(~/.zfunc $fpath) and autoload -U compinit && compinit are set in your ~/.zshrc).

```bash
#Bash
som completion bash > /etc/bash_completion.d/som
```

```bash
#Fish
som completion fish > ~/.config/fish/completions/som.fish
```

## YouTube Stability & Cookies

YouTube occasionally 403s (`HTTP Error 403: Forbidden`) for datacenter/VPN IPs — typically when the machine egresses through a CDN multi-IP range that doesn't match the `ip=` bound to the googlevideo token. SOM handles this in three layers:

1. **Client fallback chain** — every yt-dlp call tries the configured player clients in order (`web_embedded → web → tv_embedded → android → mweb → ios`). On a 403 / format error it moves to the next client and **remembers the working one** for the rest of the session. Override with `SOM_YTDLP_CLIENTS=web_embedded,android`.
2. **Optional cookies** — a Netscape `cookies.txt` exported from a logged-in browser session fixes IP-bound restrictions and age-gated content. Set `SOM_YTDLP_COOKIES=/path/to/cookies.txt` (export it with e.g. the *Get cookies.txt LOCALLY* browser extension; re-export when the session expires).
3. **Extra args** — `SOM_YTDLP_ARGS="--proxy socks5://localhost:9050"` appends verbatim to every yt-dlp command for advanced setups.

If a 403 still slips through, SOM prints a hint to run `som --update-ytdlp`, which updates the bundled yt-dlp binary.


---

## API Endpoints

| Method | Endpoint | Params | Description |
|--------|----------|--------|-------------|
| `GET` | `/api/v1/search` | `q` — search keyword | Search YouTube for tracks |
| `GET` | `/api/v1/stream` | `id` — YouTube video ID | Proxy audio stream |
| `GET` | `/api/v1/lyrics` | `id` — YouTube video ID | Fetch synced lyrics |
| `GET` | `/api/v1/resolve` | `id` — YouTube video ID | Resolve stream URL |
| `GET` | `/health` | — | Health check |

---
##  Tech Stack

### Backend
- **Go** — Fast, compiled HTTP server
- **Chi** — Lightweight HTTP router
- **yt-dlp** — YouTube audio extraction (opus format)
- **LRCLib** — Synced lyrics database
- **Bubble Tea** — TUI framework powering the terminal app
- **oto** — Cross-platform Go audio playback (decodes via `ffmpeg`)
- **SQLite** — Local persistence via `modernc.org/sqlite` (pure Go, no CGO) with WAL mode and FTS5 full-text search

### Frontend
- **React Native** — Cross-platform mobile framework
- **Expo** — Managed workflow & native modules
- **expo-av** — Audio playback engine
- **expo-media-control** — System media controls (notification & lock screen)
- **React Navigation** — Screen navigation
- **AsyncStorage** — Local data persistence

---

## License

This project is for **personal/educational use only**. It relies on YouTube content and should not be used for commercial purposes.

---

<p align="center">
  Made by <b>ミＧＩＡＮ4０４シ</b>
</p>