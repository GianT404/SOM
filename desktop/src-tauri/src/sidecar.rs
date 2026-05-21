use std::{
    net::TcpListener,
    path::PathBuf,
    sync::Mutex,
    thread,
    time::{Duration, Instant},
};

use tauri::{AppHandle, Manager, Runtime, State};
use tauri_plugin_shell::{process::CommandChild, ShellExt};

#[derive(Default)]
pub struct BackendState {
    pub url: Mutex<Option<String>>,
    child: Mutex<Option<CommandChild>>,
}

#[tauri::command]
pub fn get_backend_url(state: State<'_, BackendState>) -> Result<String, String> {
    Ok(state
        .url
        .lock()
        .map_err(|_| "backend state lock failed")?
        .clone()
        .unwrap_or_else(|| "http://127.0.0.1:8080".to_string()))
}

pub fn start_backend<R: Runtime>(app: AppHandle<R>) -> Result<(), Box<dyn std::error::Error>> {
    let port = pick_port()?;
    let url = format!("http://127.0.0.1:{port}");
    let state = app.state::<BackendState>();

    let command = backend_command(&app)?;
    let (mut rx, child) = command
        .env("HOST", "127.0.0.1")
        .env("PORT", port.to_string())
        .spawn()?;

    tauri::async_runtime::spawn(async move {
        while let Some(event) = rx.recv().await {
            if cfg!(debug_assertions) {
                println!("backend sidecar: {event:?}");
            }
        }
    });

    *state.child.lock().map_err(|_| "backend child lock failed")? = Some(child);
    *state.url.lock().map_err(|_| "backend url lock failed")? = Some(url.clone());

    wait_for_backend(&url);
    Ok(())
}

fn backend_command<R: Runtime>(
    app: &AppHandle<R>,
) -> Result<tauri_plugin_shell::process::Command, Box<dyn std::error::Error>> {
    if cfg!(debug_assertions) {
        let dev_binary = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("binaries")
            .join(dev_sidecar_filename());
        Ok(app.shell().command(dev_binary))
    } else {
        Ok(app.shell().sidecar("binaries/som-backend")?)
    }
}

fn dev_sidecar_filename() -> &'static str {
    if cfg!(target_os = "windows") {
        "som-backend-x86_64-pc-windows-msvc.exe"
    } else {
        "som-backend-x86_64-unknown-linux-gnu"
    }
}

fn pick_port() -> Result<u16, std::io::Error> {
    if TcpListener::bind(("127.0.0.1", 8080)).is_ok() {
        return Ok(8080);
    }
    let listener = TcpListener::bind(("127.0.0.1", 0))?;
    Ok(listener.local_addr()?.port())
}

fn wait_for_backend(url: &str) {
    let health = format!("{url}/health");
    let start = Instant::now();
    while start.elapsed() < Duration::from_secs(10) {
        if reqwest::blocking::get(&health)
            .map(|response| response.status().is_success())
            .unwrap_or(false)
        {
            return;
        }
        thread::sleep(Duration::from_millis(200));
    }
    eprintln!("SOM backend did not become healthy within 10 seconds: {health}");
}

impl Drop for BackendState {
    fn drop(&mut self) {
        if let Ok(mut child) = self.child.lock() {
            if let Some(child) = child.take() {
                let _ = child.kill();
            }
        }
    }
}
