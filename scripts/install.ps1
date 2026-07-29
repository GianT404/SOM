$ErrorActionPreference = "Stop"
$Repo = "GianT404/SOM"
$DestDir = Join-Path $env:LocalAppData "Programs\som"

function Info($msg)  { Write-Host "==> $msg" -ForegroundColor Cyan }
function Warn($msg)  { Write-Host "!! $msg" -ForegroundColor Yellow }
function Err($msg)   { Write-Host "LỖI: $msg" -ForegroundColor Red; exit 1 }

function Install-Deps {
    New-Item -ItemType Directory -Force -Path $DestDir | Out-Null

    # 1. Kéo yt-dlp.exe 
    $ytDlpPath = Join-Path $DestDir "yt-dlp.exe"
    if (-not (Test-Path $ytDlpPath)) {
        Info " ..."
        Invoke-WebRequest -Uri "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe" -OutFile $ytDlpPath
    } else {
        Info "yt-dlp already "
    }

    # 2. Kéo FFmpeg (Static Build)
    $ffmpegPath = Join-Path $DestDir "ffmpeg.exe"
    if (-not (Test-Path $ffmpegPath)) {
        Info "FFmpeg Static Build (Gyan)..."
        $zipPath = Join-Path $env:TEMP "ffmpeg_static.zip"
        Invoke-WebRequest -Uri "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip" -OutFile $zipPath
        
        Info "Extract FFmpeg..."
        Expand-Archive -Path $zipPath -DestinationPath $env:TEMP -Force
        
        $extractedDir = Join-Path $env:TEMP "ffmpeg-master-latest-win64-gpl\bin"
        Move-Item -Path (Join-Path $extractedDir "ffmpeg.exe") -Destination $DestDir -Force
        Move-Item -Path (Join-Path $extractedDir "ffprobe.exe") -Destination $DestDir -Force
        
        Remove-Item -Path $zipPath -Force
        Remove-Item -Path (Join-Path $env:TEMP "ffmpeg-master-latest-win64-gpl") -Recurse -Force
    }
}

function Install-Som {
    $arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { Err "Chỉ hỗ trợ amd64" }
    $asset = "som-windows-$arch.exe"
    $url = "https://github.com/$Repo/releases/latest/download/$asset"
    $dest = Join-Path $DestDir "som.exe"

    Info "Download SOM ($asset)..."
    try {
        Invoke-WebRequest -Uri $url -OutFile $dest
    } catch {
        Err "Faile ."
    }

    Info "Installed $dest"
    
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$DestDir*") {
        Info "Add $DestDir in PATH..."
        $newPath = if ([string]::IsNullOrEmpty($userPath)) { $DestDir } else { "$userPath;$DestDir" }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    }
}

Install-Deps
Install-Som