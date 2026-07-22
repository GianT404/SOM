set -euo pipefail

REPO="GianT404/SOM"
INSTALL_DIR="/usr/local/bin"

info()  { printf '\033[1;34m==>\033[0m %s\n' "$1"; }
warn()  { printf '\033[1;33m!!\033[0m %s\n' "$1"; }
error() { printf '\033[1;31mLỖI:\033[0m %s\n' "$1" >&2; exit 1; }

install_deps() {
	if command -v yt-dlp >/dev/null 2>&1 && command -v ffprobe >/dev/null 2>&1 && command -v mpv >/dev/null 2>&1; then
		info "yt-dlp, ffprobe, mpv đã có sẵn — bỏ qua bước cài dependency."
		return
	fi

	if command -v pacman >/dev/null 2>&1; then
		info "Phát hiện pacman (Arch/Manjaro) — cài yt-dlp, ffmpeg, mpv..."
		sudo pacman -S --needed --noconfirm yt-dlp ffmpeg mpv

	elif command -v apt >/dev/null 2>&1; then
		info "Phát hiện apt (Debian/Ubuntu) — cài ffmpeg, mpv, yt-dlp..."
		sudo apt update
		sudo apt install -y ffmpeg mpv
		if command -v pip3 >/dev/null 2>&1; then
			pip3 install --user -U yt-dlp
		else
			sudo apt install -y yt-dlp
		fi

	elif command -v dnf >/dev/null 2>&1; then
		info "Phát hiện dnf (Fedora) — cài ffmpeg, mpv, yt-dlp..."
		sudo dnf install -y ffmpeg mpv yt-dlp

	elif command -v brew >/dev/null 2>&1; then
		info "Phát hiện Homebrew (macOS) — cài yt-dlp, ffmpeg, mpv..."
		brew install yt-dlp ffmpeg mpv

	else
		warn "Không nhận diện được package manager (pacman/apt/dnf/brew)."
		warn "Tự cài yt-dlp, ffmpeg (có ffprobe), mpv theo distro của bạn rồi chạy lại script này."
		exit 1
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
		*) error "Kiến trúc CPU không được hỗ trợ: $(uname -m) (hiện chỉ có bản amd64)" ;;
	esac

	asset="som-${os}-${arch}"
	tmp="$(mktemp)"

	info "Tải ${asset} (bản mới nhất)..."
	curl -fL "https://github.com/${REPO}/releases/latest/download/${asset}" -o "$tmp" \
		|| error "Tải thất bại — kiểm tra lại release có build cho ${asset} chưa."

	chmod +x "$tmp"

	info "Cài vào ${INSTALL_DIR}/som (cần sudo)..."
	sudo mv "$tmp" "${INSTALL_DIR}/som"
}

install_deps
install_som

echo
info "Xong! Thử chạy: som --version"