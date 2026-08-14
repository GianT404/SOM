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
		return fmt.Errorf(" %s — "+
			"please manually copy the binary to a directory in your $PATH", runtime.GOOS)
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
			return fmt.Errorf("no permission to write to %s — run again with sudo:\n  sudo som --install", destDir)
		}
		return err
	}
	fmt.Println("Installed to", dest)
	fmt.Println("You can now run `som` from anywhere (open a new terminal if you don't see the effect).")
	return nil
}

func installWindows() error {
	localAppData := os.Getenv("LocalAppData")
	if localAppData == "" {
		return fmt.Errorf("no LocalAppData environment variable found")
	}
	destDir := filepath.Join(localAppData, "Programs", "som")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create installation directory: %w", err)
	}

	dest := filepath.Join(destDir, "som.exe")
	if err := copyExecutableTo(dest); err != nil {
		return err
	}

	fmt.Println("Installed to", dest)
	fmt.Println()
	fmt.Println("To run `som` from anywhere, add the following directory to your PATH (do this only once):")
	fmt.Println("   ", destDir)
	fmt.Println()
	fmt.Println("How to add to PATH: type \"env\" in Windows Search → select \"Edit environment variables")
	fmt.Println("for your account\" → select \"User variables\" → select \"Path\" → Edit → New → paste")
	fmt.Println("the path above → OK all windows → open a new terminal to apply the changes.")
	return nil
}

func copyExecutableTo(dest string) error {
	src, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine the location of the running binary: %w", err)
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
