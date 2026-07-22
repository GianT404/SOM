package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

func runInstall() error {
	switch runtime.GOOS {
	case "linux", "darwin":
		return installUnix()
	case "windows":
		return installWindows()
	default:
		return fmt.Errorf("chưa hỗ trợ --install tự động trên %s — "+
			"tự copy binary vào 1 thư mục đã có trong $PATH nhé", runtime.GOOS)
	}
}

func alreadyInstalled() bool {
	exe, err := os.Executable()
	if err != nil {
		return true
	}
	switch runtime.GOOS {
	case "linux", "darwin":
		return exe == "/usr/local/bin/som"
	case "windows":
		localAppData := os.Getenv("LocalAppData")
		if localAppData == "" {
			return true
		}
		return exe == filepath.Join(localAppData, "Programs", "som", "som.exe")
	default:
		return true
	}
}

func installUnix() error {
	const destDir = "/usr/local/bin"
	dest := filepath.Join(destDir, "som")

	if err := copyExecutableTo(dest); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("không đủ quyền ghi vào %s — chạy lại với sudo:\n  sudo som --install", destDir)
		}
		return err
	}
	fmt.Println("Đã cài vào", dest)
	fmt.Println("Giờ có thể gõ `som` ở bất kỳ đâu (mở terminal mới nếu chưa thấy tác dụng).")
	return nil
}

func installWindows() error {
	localAppData := os.Getenv("LocalAppData")
	if localAppData == "" {
		return fmt.Errorf("không tìm thấy biến môi trường LocalAppData")
	}
	destDir := filepath.Join(localAppData, "Programs", "som")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("tạo thư mục cài đặt thất bại: %w", err)
	}

	dest := filepath.Join(destDir, "som.exe")
	if err := copyExecutableTo(dest); err != nil {
		return err
	}

	fmt.Println("Đã cài vào", dest)
	fmt.Println()
	fmt.Println("Để gõ `som` được ở bất kỳ đâu, thêm thư mục sau vào PATH (chỉ cần làm 1 lần):")
	fmt.Println("   ", destDir)
	fmt.Println()
	fmt.Println("Cách thêm: gõ \"env\" vào Windows Search → chọn \"Edit environment variables")
	fmt.Println("for your account\" → mục \"User variables\" → chọn \"Path\" → Edit → New → dán")
	fmt.Println("đường dẫn ở trên → OK hết các cửa sổ → mở terminal mới để áp dụng.")
	return nil
}

func copyExecutableTo(dest string) error {
	src, err := os.Executable()
	if err != nil {
		return fmt.Errorf("không xác định được vị trí binary đang chạy: %w", err)
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(0o755)
}
