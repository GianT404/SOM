<p align="center">
  <img src="./assets/logo.svg" alt="SOM Logo" width="120" />
</p>

<h1 align="center">SOM</h1>

<p align="center">
  <b>Stream and play local music</b>
  <br>
  <b>Linux, Windows, Android, iOS</b>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/React_Native-0.83-61DAFB?logo=react&logoColor=white" />
  <img src="https://img.shields.io/badge/Expo-55-000020?logo=expo&logoColor=white" />
</p>

<p align="center">
  <img src="./assets/thumbnail.png" alt="SOM" width="100%" />
</p>

## Demo

https://github.com/user-attachments/assets/b2050ea6-6621-4a2c-a366-9326a810218e

## Audio Visualizer

https://github.com/user-attachments/assets/d7bf017b-7a73-4f7e-8d07-964e5f460249

---

## Features

| Feature | Description |
|---------|-------------|
| Search & Stream | YouTube search, stream audio without full download |
| Offline Playback | Download tracks (`.opus`) and support for additional audio formats (`.mp3`, `.mp4`, `.flac`, `.m4a`, `.wav`, `.aac`, `.ogg`, `.webm`, `.wma`,`.aiff`,`.alac`) |
| Playlists | Create, manage, and play custom playlists |
| Synced Lyrics | LRCLib + YouTube subtitles fallback, multi-language, millisecond-precise seeking |
| Audio Presets | Bass Boost, Nightcore, Daycore, Lo-Fi via command menu |
| Media Controls | Lock screen & notification controls (play, pause, skip, seek) |
| Shuffle | Smart shuffle — avoids replaying recent tracks and back-to-back same artist |
| Audio Visualizer | Live 2D bar / 3D wireframe driven by real-time system audio (`\`) |
| Theme | Default (accent colors) or Mono (all-white text), configurable in Settings |
| Mouse Support | Click sidebar, double-click to play, wheel scroll, progress bar seek |
| Gapless Playback | Pre-decodes next track when current has <3s remaining |
| Resilient YouTube | Automatic yt-dlp client fallback against 403s |
| Self Install/Update | `som --install`, `som --upgrade`, minisign signature verification |
| CLI & Completion | Cobra with auto-completion for Zsh, Bash, Fish |
| AVRCP/MPRIS2 | Bluetooth media controls on Linux desktop |

---

## Quick Start

### Prerequisites

- **Go** 1.26+
- **yt-dlp** and **ffmpeg** in `$PATH`
- **Node.js** 18+ & **npm** (for mobile app)

### Install

```bash
# Linux
curl -fsSL https://raw.githubusercontent.com/GianT404/SOM/main/scripts/install.sh | bash
```


```powershell
# Windows
curl.exe -O https://raw.githubusercontent.com/GianT404/SOM/main/scripts/install.ps1 | iex
```

### Build TUI

```bash
git clone https://github.com/GianT404/SOM.git && cd SOM
go build -o som ./cmd/som
./som
```

### Build Backend

```bash
go build -o server ./cmd/server
./server
```

Default port `8080`. Configure via env vars:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `HOST` | (all) | Bind host |
| `SOM_DOWNLOAD_DIR` | `~/.local/share/som` | Download directory |
| `SOM_YTDLP_COOKIES` | — | Netscape `cookies.txt` for age-gated/restricted content |
| `SOM_YTDLP_ARGS` | — | Extra yt-dlp args (e.g. `--proxy socks5://localhost:9050`) |
| `SOM_YTDLP_CLIENTS` | `web_embedded,web,tv_embedded,android,mweb,ios` | yt-dlp client fallback chain |

### Build Mobile App

```bash
cd app && npm install --legacy-peer-deps
npx expo start
```

---

### TUI Key Bindings

| Key | Action |
|-----|--------|
| `?` | Help popup |
| `Tab` / `1`-`7` | Sidebar tabs |
| `/` | Focus search input |
| `:` | Command popup (add to queue,audio settings, playback, rename, delete, move to playlist, sort, file info) |
| `\` | Audio visualizer (press `l` to toggle 2D/3D) |
| `Enter` | Play selected track |
| `Space` | Play / pause |
| `]` / `[` | Next / previous |
| `r` | Toggle shuffle |
| `←` / `→` | Seek -/+ 5s |
| `+` / `-` | Volume |
| `l` | Choose lyrics language |
| `pgup` / `pgdown` | Scroll lyrics |
| `d` | Download selected track |
| `Delete` | Remove playlist|
| `Esc` | Settings popup |
| `alt+q` | Quit |

### Theme

Configurable in **Settings** (`Esc`). Two modes:

- **Default** — accent color on borders, selected items, and progress bar
- **Mono** — all-white text and borders, no accent colors (high-contrast)

Selected theme persists across sessions.

### Mouse Support

Configurable in **Settings** (`Esc`), default OFF.


### Logs

Logs tab captures all `log` output in-memory (last 2000 lines). Never written to disk during normal operation. On crash/panic, dumped to `~/crash_som_tui_<timestamp>.log`.

---

## Shell Completion

```bash
# Zsh
mkdir -p ~/.zfunc
som completion zsh > ~/.zfunc/_som
# Add to ~/.zshrc: fpath=(~/.zfunc $fpath) && autoload -U compinit && compinit

# Bash
som completion bash > /etc/bash_completion.d/som

# Fish
som completion fish > ~/.config/fish/completions/som.fish
```

---

## CLI Flags

| Flag | Description |
|------|-------------|
| `--server <URL>` | Remote mode — point to a SOM backend |
| `--api-key <KEY>` | API key for remote mode (or `SOM_API_KEY` env) |
| `--download-dir` | Override download directory |
| `--install` | Copy binary to `/usr/local/bin` |
| `--upgrade` | Download latest release (minisign verified) |
| `--check-update` | Check for updates without installing |
| `--uninstall` | Remove installed binary |
| `--update-ytdlp` | Update bundled yt-dlp |
| `--version` | Print version |
| `--changelog` | Print current version commits |

---

## YouTube 403 Fix

If streaming fails with `403 Forbidden`:

1. Update yt-dlp: `som --update-ytdlp`
2. Export cookies from a logged-in browser (`Get cookies.txt LOCALLY` extension) → set `SOM_YTDLP_COOKIES=/path/to/cookies.txt`
3. Override client chain: `SOM_YTDLP_CLIENTS=web_embedded,android`
4. Use proxy: `SOM_YTDLP_ARGS="--proxy socks5://localhost:9050"`

---

## API

| Endpoint | Params | Description |
|----------|--------|-------------|
| `GET /api/v1/search` | `q` | Search YouTube |
| `GET /api/v1/stream` | `id` | Proxy audio stream |
| `GET /api/v1/lyrics` | `id` | Fetch synced lyrics |
| `GET /api/v1/resolve` | `id` | Resolve stream URL |
| `GET /health` | — | Health check |

---

## License

Personal/educational use only. Not for commercial purposes.

---

<p align="center">
  Made by <b>ミＧＩＡＮ4０４シ</b>
</p>
