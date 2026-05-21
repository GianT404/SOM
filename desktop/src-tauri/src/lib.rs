mod downloads;
mod sidecar;

use downloads::{download_track, read_downloaded_track, reveal_downloads_dir};
use sidecar::{get_backend_url, start_backend};

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(sidecar::BackendState::default())
        .setup(|app| {
            if std::env::var("SOM_DESKTOP_EXTERNAL_BACKEND").is_err() {
                start_backend(app.handle().clone())?;
            }
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            get_backend_url,
            download_track,
            read_downloaded_track,
            reveal_downloads_dir
        ])
        .run(tauri::generate_context!())
        .expect("error while running SOM desktop");
}
