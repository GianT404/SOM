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
		info "Downloading yt-dlp..."
		sudo curl -L "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp" -o /usr/local/bin/yt-dlp
		sudo chmod a+rx /usr/local/bin/yt-dlp
	else
		warn "yt-dlp đã có — bỏ qua tự cập nhật (lệnh 'yt-dlp -U' hay bị GitHub rate-limit; muốn cập nhật thì chạy riêng: yt-dlp -U)"
	fi

	# 2. Kéo FFmpeg Static Build
	if ! command -v ffmpeg >/dev/null 2>&1; then
		info "Downloading FFmpeg static build..."
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
		info "FFmpeg already."
	fi
}

latest_release_with_asset() {
	local name="$1"
	curl -fsSL "https://github.com/$REPO/releases.atom" |
		grep -oE 'releases/tag/[A-Za-z0-9._/-]+' |
		sed 's|releases/tag/||' |
		while IFS= read -r tag; do
			if curl -fsIL -o /dev/null "https://github.com/$REPO/releases/download/$tag/$name" 2>/dev/null; then
				printf '%s\n' "$tag"
				break
			fi
		done
}

install_som() {
	local os arch asset tag tmp url
	case "$(uname -s)" in
		Linux)  os="linux" ;;
		Darwin) os="darwin" ;;
		*) error "Unsupported operating system: $(uname -s)" ;;
	esac
	case "$(uname -m)" in
		x86_64|amd64) arch="amd64" ;;
		arm64|aarch64) arch="arm64" ;;
		*) error "CPU architecture not supported: $(uname -m)" ;;
	esac

	asset="som-${os}-${arch}"
	info "Tìm bản mới nhất của ($asset)..."

	tag="$(latest_release_with_asset "$asset")"
	if [ -z "$tag" ]; then
		error "Không tìm thấy release nào có asset $asset (workflow TUI chưa build cho platform này?)"
	fi

	tmp="$(mktemp -d)"
	url="https://github.com/$REPO/releases/download/$tag/$asset"
	info "($asset) ← $tag"
	curl -fsSL "$url" -o "$tmp/som"
	chmod +x "$tmp/som"

	info "Install $INSTALL_DIR/som ..."
	sudo mv "$tmp/som" "$INSTALL_DIR/som"
	rm -rf "$tmp"
}

install_deps
install_som
