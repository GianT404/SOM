use std::{
    fs,
    io::{Read, Write},
    path::PathBuf,
};

use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager, Runtime};

#[derive(Debug, Deserialize)]
pub struct DownloadTrackPayload {
    id: String,
    title: String,
    uploader: String,
    thumbnail: String,
    duration: f64,
    stream_url: String,
    lyrics: Option<serde_json::Value>,
}

#[derive(Debug, Serialize)]
pub struct OfflineTrack {
    id: String,
    title: String,
    uploader: String,
    thumbnail: String,
    duration: f64,
    local_path: String,
    downloaded_at: u64,
    lyrics: Option<serde_json::Value>,
}

#[tauri::command]
pub fn download_track<R: Runtime>(
    app: AppHandle<R>,
    track: DownloadTrackPayload,
) -> Result<OfflineTrack, String> {
    let downloads_dir = downloads_dir(&app)?;
    fs::create_dir_all(&downloads_dir).map_err(|err| err.to_string())?;

    let target = downloads_dir.join(format!("{}.m4a", sanitize_filename(&track.id)));
    let mut response = reqwest::blocking::get(&track.stream_url).map_err(|err| err.to_string())?;
    if !response.status().is_success() {
        return Err(format!("download failed with status {}", response.status()));
    }

    let mut file = fs::File::create(&target).map_err(|err| err.to_string())?;
    response.copy_to(&mut file).map_err(|err| err.to_string())?;
    file.flush().map_err(|err| err.to_string())?;
    drop(file);

    if !is_m4a_file(&target) {
        let _ = fs::remove_file(&target);
        return Err("downloaded file is not a playable m4a audio file".to_string());
    }

    Ok(OfflineTrack {
        id: track.id,
        title: track.title,
        uploader: track.uploader,
        thumbnail: track.thumbnail,
        duration: track.duration,
        local_path: target.to_string_lossy().to_string(),
        downloaded_at: now_ms(),
        lyrics: track.lyrics,
    })
}

#[tauri::command]
pub fn reveal_downloads_dir<R: Runtime>(app: AppHandle<R>) -> Result<String, String> {
    let path = downloads_dir(&app)?;
    fs::create_dir_all(&path).map_err(|err| err.to_string())?;
    Ok(path.to_string_lossy().to_string())
}

#[tauri::command]
pub fn read_downloaded_track<R: Runtime>(
    app: AppHandle<R>,
    local_path: String,
) -> Result<Vec<u8>, String> {
    let downloads_dir = downloads_dir(&app)?;
    let path = PathBuf::from(local_path);
    let canonical_downloads = downloads_dir.canonicalize().map_err(|err| err.to_string())?;
    let canonical_path = path.canonicalize().map_err(|err| err.to_string())?;

    if !canonical_path.starts_with(canonical_downloads) {
        return Err("track is outside the downloads directory".to_string());
    }

    if !is_m4a_file(&canonical_path) {
        return Err("downloaded file is not a playable m4a audio file".to_string());
    }

    fs::read(canonical_path).map_err(|err| err.to_string())
}

fn downloads_dir<R: Runtime>(app: &AppHandle<R>) -> Result<PathBuf, String> {
    app.path()
        .app_data_dir()
        .map(|path| path.join("downloads"))
        .map_err(|err| err.to_string())
}

fn sanitize_filename(value: &str) -> String {
    value
        .chars()
        .filter(|ch| ch.is_ascii_alphanumeric() || matches!(ch, '-' | '_'))
        .collect::<String>()
}

fn now_ms() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|duration| duration.as_millis() as u64)
        .unwrap_or_default()
}

fn is_m4a_file(path: &PathBuf) -> bool {
    let Ok(mut file) = fs::File::open(path) else {
        return false;
    };

    let mut header = [0u8; 12];
    file.read_exact(&mut header).is_ok() && &header[4..8] == b"ftyp"
}
