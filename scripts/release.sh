# ./scripts/release.sh v0.x.x "cmt"
set -euo pipefail

if [ $# -lt 1 ]; then
	echo "Dùng: ./scripts/release.sh <version> [commit message]"
	echo "Ví dụ: ./scripts/release.sh v0.1.2 \"sửa lỗi mpv\""
	exit 1
fi

VERSION="$1"
MSG="${2:-release ${VERSION}}"

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "LỖI: version phải theo dạng vX.Y.Z (ví dụ v0.1.2), bạn đưa: ${VERSION}"
	exit 1
fi

if git rev-parse "$VERSION" >/dev/null 2>&1; then
	echo "LỖI: tag ${VERSION} đã tồn tại rồi, dùng số khác."
	exit 1
fi

# Chỉ commit nếu thực sự có gì thay đổi (tránh lỗi "nothing to commit")
if ! git diff --quiet || ! git diff --cached --quiet || [ -n "$(git status --porcelain)" ]; then
	echo "==> Commit thay đổi hiện tại..."
	git add .
	git commit -m "$MSG"
else
	echo "==> Không có gì thay đổi để commit, bỏ qua bước này."
fi

echo "==> Push code lên nhánh hiện tại..."
git push

echo "==> Tạo tag ${VERSION}..."
git tag "$VERSION"

echo "==> Push tag..."
git push origin "$VERSION"

echo
echo "Xong!"