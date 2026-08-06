$ErrorActionPreference = "Stop"
$Repo = "GianT404/SOM"
$DestDir = Join-Path $env:LocalAppData "Programs\som"

function Info($msg) { Write-Host "==> $msg" -ForegroundColor Cyan }
function Warn($msg) { Write-Host "!! $msg" -ForegroundColor Yellow }
function Err($msg)  { Write-Host "ERROR: $msg" -ForegroundColor Red; exit 1 }

function Install-Deps {
    New-Item -ItemType Directory -Force -Path $DestDir | Out-Null

    # 1. Fetch yt-dlp.exe
    $ytDlpPath = Join-Path $DestDir "yt-dlp.exe"
    if (-not (Test-Path $ytDlpPath)) {
        Info "Downloading yt-dlp..."
        Invoke-WebRequest -Uri "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe" -OutFile $ytDlpPath
    } else {
        Info "yt-dlp already installed."
    }

    # 2. Fetch FFmpeg static build
    $ffmpegPath = Join-Path $DestDir "ffmpeg.exe"
    if (-not (Test-Path $ffmpegPath)) {
        Info "Downloading FFmpeg static build (Gyan)..."
        $zipPath = Join-Path $env:TEMP "ffmpeg_static.zip"
        Invoke-WebRequest -Uri "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip" -OutFile $zipPath

        Info "Extracting FFmpeg..."
        Expand-Archive -Path $zipPath -DestinationPath $env:TEMP -Force

        $extractedDir = Join-Path $env:TEMP "ffmpeg-master-latest-win64-gpl\bin"
        Move-Item -Path (Join-Path $extractedDir "ffmpeg.exe") -Destination $DestDir -Force
        Move-Item -Path (Join-Path $extractedDir "ffprobe.exe") -Destination $DestDir -Force

        Remove-Item -Path $zipPath -Force
        Remove-Item -Path (Join-Path $env:TEMP "ffmpeg-master-latest-win64-gpl") -Recurse -Force
    }
}

# Find the newest release that actually contains an asset named <name>.
# Avoids the GitHub API (rate-limited, 403) by reading the releases.atom feed
# and probing each tag's asset URL with a HEAD request.
function Get-LatestReleaseWithAsset {
    param([string]$AssetName)

    $feed = Invoke-RestMethod -Uri "https://github.com/$Repo/releases.atom"
    foreach ($entry in @($feed.entry)) {
        $tag = $null
        foreach ($link in @($entry.link)) {
            if ($link.href -match '/releases/tag/([^/]+)/?$') {
                $tag = $Matches[1]
                break
            }
        }
        if (-not $tag) { continue }

        $url = "https://github.com/$Repo/releases/download/$tag/$AssetName"
        try {
            Invoke-WebRequest -Uri $url -Method Head -UseBasicParsing | Out-Null
            return $tag
        } catch {
            # Asset not in this release - try the next one.
        }
    }
    return $null
}

function Install-Som {
    if (-not [Environment]::Is64BitOperatingSystem) {
        Err "Only 64-bit Windows is supported."
    }
    $asset = "som-windows-amd64.exe"

    Info "Looking for the latest build of ($asset)..."
    $tag = Get-LatestReleaseWithAsset -AssetName $asset
    if (-not $tag) {
        Err "No release found containing asset $asset (TUI workflow not built for this platform?)"
    }

    $url = "https://github.com/$Repo/releases/download/$tag/$asset"
    $dest = Join-Path $DestDir "som.exe"

    Info "($asset) <- $tag"
    Invoke-WebRequest -Uri $url -OutFile $dest

    Info "Installed $dest"

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$DestDir*") {
        Info "Adding $DestDir to PATH..."
        $newPath = if ([string]::IsNullOrEmpty($userPath)) { $DestDir } else { "$userPath;$DestDir" }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    }

    # Warn if an older som earlier in PATH shadows the one we just installed.
    $cmd = Get-Command som -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($cmd -and $cmd.CommandType -eq "Application" -and $cmd.Source -ne $dest) {
        Warn "PATH prefers '$($cmd.Source)' (an old copy) over the new $dest."
        Warn "Remove the old copy to use the new one:  Remove-Item '$($cmd.Source)'"
    }
}

Install-Deps
Install-Som
