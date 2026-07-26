#!/usr/bin/env bash
set -euo pipefail

REPO="GianT404/SOM"
INSTALL_DIR="/usr/local/bin"

info()  { printf '\033[1;34m==>\033[0m %s\n' "$1"; }
warn()  { printf '\033[1;33m!!\033[0m %s\n' "$1"; }
error() { printf '\033[1;31mLỖI:\033[0m %s\n' "$1" >&2; exit 1; }

install_deps() {
	# 1. Kéo yt-dlp
	if ! command -v yt-dlp >/dev/null 2>&1; then
		info "Đang kéo yt-dlp mới nhất trực tiếp từ GitHub..."
		sudo curl -L "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp" -o /usr/local/bin/yt-dlp
		sudo chmod a+rx /usr/local/bin/yt-dlp
	else
		info "yt-dlp đã tồn tại, tiến hành tự cập nhật..."
		sudo yt-dlp -U || true
	fi

	# 2. Kéo FFmpeg Static Build
	if ! command -v ffmpeg >/dev/null 2>&1; then
		info "Đang kéo FFmpeg Static Build..."
		if [ "$(uname -s)" = "Linux" ]; then
			curl -L "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz" -o /tmp/ffmpeg.tar.xz
			tar -xf /tmp/ffmpeg.tar.xz -C /tmp
			sudo mv /tmp/ffmpeg-*-static/ffmpeg /tmp/ffmpeg-*-static/ffprobe /usr/local/bin/
			rm -rf /tmp/ffmpeg*
		elif [ "$(uname -s)" = "Darwin" ]; then
			curl -L "https://evermeet.cx/ffmpeg/ffmpeg-115312-g6c039750d7.zip" -o /tmp/ffmpeg.zip
			unzip -q /tmp/ffmpeg.zip -d /tmp/
			sudo mv /tmp/ffmpeg /usr/local/bin/
			rm /tmp/ffmpeg.zip
		fi
	else
		info "FFmpeg đã có sẵn."
	fi
}

install_som() {
	local os arch asset tmp
	case "$(uname -s)" in
		Linux)  os="linux" ;;
		Darwin) os="darwin" ;;
		*) error "Hệ điều hành không được hỗ trợ: $(uname -s)" ;;
	esac
	case "$(uname -m)" in
		x86_64|amd64) arch="amd64" ;;
		arm64|aarch64) arch="arm64" ;;
		*) error "Kiến trúc CPU không hỗ trợ: $(uname -m)" ;;
	esac

	asset="som-${os}-${arch}"
	tmp="$(mktemp -d)"
	
	info "Tải SOM bản mới nhất từ GitHub ($asset)..."
	curl -sL "https://github.com/$REPO/releases/latest/download/$asset" -o "$tmp/som"
	chmod +x "$tmp/som"

	info "Cài đặt vào $INSTALL_DIR/som ..."
	sudo mv "$tmp/som" "$INSTALL_DIR/som"
	rm -rf "$tmp"

	info "Cài đặt thành công! Gõ 'som' để quẩy nhạc."
}

install_deps
install_som