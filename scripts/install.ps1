ErrorActionPreference = "Stop"

$Repo = "GianT404/SOM"

function Info($msg)  { Write-Host "==> $msg" -ForegroundColor Cyan }
function Warn($msg)  { Write-Host "!! $msg" -ForegroundColor Yellow }
function Err($msg)   { Write-Host "LOI: $msg" -ForegroundColor Red; exit 1 }

function Install-Deps {
    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
        Warn "Khong tim thay winget. Can Windows 10 1809+ / Windows 11, hoac cai"
        Warn "'App Installer' tu Microsoft Store roi chay lai script nay."
        exit 1
    }

    Info "Cai yt-dlp..."
    winget install --id yt-dlp.yt-dlp --silent --accept-package-agreements --accept-source-agreements

    Info "Cai ffmpeg (kem ffprobe)..."
    winget install --id Gyan.FFmpeg --silent --accept-package-agreements --accept-source-agreements

    Info "Cai mpv..."
    winget install --id mpv-player.mpv --silent --accept-package-agreements --accept-source-agreements

    Warn "Neu mpv/ffmpeg vua cai xong ma 'som' van bao khong tim thay, mo lai"
    Warn "terminal moi (winget can terminal moi de nhan PATH vua duoc cap nhat)."
}

function Install-Som {
    $arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { Err "Chi ho tro amd64" }
    $asset = "som-windows-$arch.exe"
    $url = "https://github.com/$Repo/releases/latest/download/$asset"

    $destDir = Join-Path $env:LocalAppData "Programs\som"
    New-Item -ItemType Directory -Force -Path $destDir | Out-Null
    $dest = Join-Path $destDir "som.exe"

    Info "Tai $asset (ban moi nhat)..."
    try {
        Invoke-WebRequest -Uri $url -OutFile $dest
    } catch {
        Err "Tai that bai - kiem tra lai release co build cho $asset chua."
    }

    Info "Da cai vao $dest"
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$destDir*") {
        Info "Them $destDir vao PATH cua user..."
        $newPath = if ([string]::IsNullOrEmpty($userPath)) { $destDir } else { "$userPath;$destDir" }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Warn "Da them vao PATH - mo terminal MOI de dung duoc lenh 'som' ngay."
    } else {
        Info "$destDir da co san trong PATH."
    }
}

Install-Deps
Install-Som

Write-Host ""
Info "Xong! Mo terminal moi roi thu: som --version"