#!/usr/bin/env bash
# ./scripts/release-mobile.sh v0.x.x "cmt" — chỉ build/release APK mobile
# (chạy lệnh cần thiết, không cần sudo)
set -euo pipefail

if [ $# -lt 1 ]; then
	echo "Dùng: ./scripts/release-mobile.sh <version> [commit message]"
	echo "Ví dụ: ./scripts/release-mobile.sh v0.1.2 \"thêm màn hình lyrics\""
	exit 1
fi

VERSION="$1"
MSG="${2:-release mobile ${VERSION}}"
TAG="app-${VERSION}"

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "LỖI: version phải theo dạng vX.Y.Z (ví dụ v0.1.2), bạn đưa: ${VERSION}"
	exit 1
fi

if git rev-parse "$TAG" >/dev/null 2>&1; then
	echo "LỖI: tag ${TAG} đã tồn tại rồi, dùng số khác."
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

echo "==> Tạo tag ${TAG} (chỉ trigger workflow mobile)..."
git tag "$TAG"

echo "==> Push tag..."
git push origin "$TAG"

echo
echo "Xong! Tag ${TAG} đã kích hoạt release-som-mobile.yml (APK)."
