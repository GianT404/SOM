$ErrorActionPreference = "Stop"

Write-Host "Bắt đầu quá trình cài đặt SOM..."

Write-Host "Đang cài đặt dependencies (yt-dlp, ffmpeg, mpv)..."
winget install yt-dlp.yt-dlp --accept-package-agreements --accept-source-agreements
winget install Gyan.FFmpeg --accept-package-agreements --accept-source-agreements
winget install mpv-player.mpv --accept-package-agreements --accept-source-agreements

$Repo = "gian404/som"
$ReleaseUrl = "https://github.com/$Repo/releases/latest/download/som-windows-amd64.exe"
$InstallDir = "$env:USERPROFILE\.local\bin"

if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
}

$DestPath = "$InstallDir\som.exe"
Write-Host "Đang tải binary về: $DestPath"
Invoke-WebRequest -Uri $ReleaseUrl -OutFile $DestPath

$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "Đã thêm $InstallDir vào User PATH."
}

Write-Host "Cài đặt thành công. Vui lòng khởi động lại terminal để áp dụng lệnh 'som'."