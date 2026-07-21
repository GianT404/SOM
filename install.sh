#!/bin/bash

# Dừng script ngay lập tức nếu có bất kỳ lệnh nào trả về lỗi
set -e

echo "Bắt đầu quá trình cài đặt SOM..."

# 1. Cài đặt Dependencies
echo "Đang kiểm tra và cài đặt dependencies (yt-dlp, ffmpeg, mpv)..."
if command -v pacman &> /dev/null; then
    sudo pacman -S --noconfirm --needed yt-dlp ffmpeg mpv
elif command -v apt &> /dev/null; then
    sudo apt update && sudo apt install -y yt-dlp ffmpeg mpv
else
    echo "Lỗi: Không hỗ trợ trình quản lý gói hiện tại. Vui lòng cài đặt yt-dlp, ffmpeg và mpv theo cách thủ công."
    exit 1
fi

# 2. Xác định OS và Architecture
REPO="gian404/som"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
if [ "$ARCH" = "x86_64" ]; then ARCH="amd64"; fi

DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/som-$OS-$ARCH"

# 3. Tải và phân quyền
echo "Đang tải bản release mới nhất từ: $DOWNLOAD_URL"
curl -fSL "$DOWNLOAD_URL" -o /tmp/som

echo "Đang cấu hình phân vùng thực thi (yêu cầu quyền sudo)..."
chmod +x /tmp/som
sudo mv /tmp/som /usr/local/bin/som

echo "Cài đặt thành công. Bạn có thể sử dụng lệnh 'som' trên terminal."