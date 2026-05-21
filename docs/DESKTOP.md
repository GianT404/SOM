# SOM Desktop

SOM desktop is a separate Tauri v2 app in `desktop/`. It does not replace or import the Expo runtime used by `app/`.

## Architecture

- Frontend: React + Vite + TypeScript in `desktop/src`.
- Shell: Tauri v2 in `desktop/src-tauri`.
- Backend: existing Go server from `cmd/server`, bundled as a Tauri sidecar.
- Audio: desktop uses `HTMLAudioElement` against the same `/api/v1/stream` endpoint.
- Lyrics: desktop calls `/api/v1/lyrics` and syncs lines against player position.
- Offline library: desktop downloads audio through a Tauri command into the app data directory, then stores playlist metadata in localStorage.

The mobile app remains in `app/` and keeps using Expo, `expo-av`, AsyncStorage, and `expo-file-system`.

## Backend Sidecar

The Go backend still defaults to `PORT=8080` and `YTDLP_PATH=yt-dlp`.

For desktop, Tauri starts the sidecar with:

- `HOST=127.0.0.1`
- `PORT=8080` if free, otherwise a free localhost port

The frontend gets the base URL from the Tauri command `get_backend_url`; it does not hardcode the production sidecar port. If `SOM_DESKTOP_EXTERNAL_BACKEND=1` is set, Tauri skips spawning the sidecar and the frontend falls back to `http://127.0.0.1:8080` for development.

## Linux Dev

Install prerequisites on Arch Linux/GNOME:

```bash
sudo pacman -S --needed base-devel curl wget file openssl webkit2gtk-4.1 libayatana-appindicator gtk3 librsvg rust go nodejs npm yt-dlp
```

Install JS dependencies from the repo root:

```bash
npm install
```

Run desktop dev:

```bash
npm run desktop:dev
```

This builds `desktop/src-tauri/binaries/som-backend-x86_64-unknown-linux-gnu`, starts Vite, starts Tauri, and launches the Go backend sidecar.

## Linux Build

```bash
npm run desktop:build:linux
```

Expected outputs:

- `desktop/src-tauri/target/release/bundle/deb/*.deb`

The default Linux build creates a `.deb` because AppImage bundling may need to download `linuxdeploy` from GitHub. To try both AppImage and `.deb`:

```bash
npm run desktop:build:linux:appimage
```

Expected AppImage output:

- `desktop/src-tauri/target/release/bundle/appimage/*.AppImage`

Tauri packaging may fail if system WebKit/AppIndicator packages are missing. Install the Arch packages above and retry. AppImage packaging may also fail if GitHub downloads are blocked.

## Windows Build

Build Windows on Windows or in a Windows CI runner:

```powershell
npm install
npm run desktop:build:windows
```

Windows requirements:

- Rust stable MSVC toolchain
- Microsoft C++ Build Tools
- WebView2 Runtime
- Go
- Node.js/npm
- `yt-dlp.exe` available in `PATH` or configured through `YTDLP_PATH`

The Windows sidecar binary is prepared as:

```text
desktop/src-tauri/binaries/som-backend-x86_64-pc-windows-msvc.exe
```

Linux-to-Windows cross packaging is intentionally not required because Tauri Windows bundling is more reliable on a Windows runner.

## Useful Scripts

```bash
npm run desktop:build:sidecar:linux
npm run desktop:build:sidecar:windows
npm run desktop:build:frontend
npm run desktop:build
```

## Manual Test Checklist

- Start `npm run desktop:dev` on Linux.
- Confirm Settings shows `Local backend`.
- Search for a song from the Search screen.
- Play a result and confirm the bottom player updates.
- Pause/resume with the play button and Space when focus is not in the search input.
- Seek with the progress slider.
- Use previous/next buttons and ArrowLeft/ArrowRight.
- Open Lyrics and confirm synced lines load or an empty/error state is shown.
- Download a track and confirm it appears in Library/Home playlist.
- Quit the Tauri app and confirm the sidecar backend process stops.
- Start the mobile app from `app/` to confirm its dependencies/imports are unchanged.

## Troubleshooting

- Missing Rust/Cargo: install Rust with `rustup` or your distro package manager.
- Missing Tauri Linux dependencies: install `webkit2gtk-4.1`, `libayatana-appindicator`, `gtk3`, `librsvg`, `openssl`, `curl`, `wget`, and `file`.
- Missing npm/pnpm: this repo uses npm scripts; install Node.js and npm.
- Port 8080 occupied: desktop will select another localhost port for sidecar production/dev. If using `SOM_DESKTOP_EXTERNAL_BACKEND=1`, make sure your own backend is on `127.0.0.1:8080`.
- Missing `yt-dlp`: install `yt-dlp` and make it executable in `PATH`, or set `YTDLP_PATH`.
- Windows WebView2/C++ errors: install WebView2 Runtime and Microsoft C++ Build Tools.
