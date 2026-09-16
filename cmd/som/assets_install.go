package main

import (
	"bytes"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

func installDesktopAssets() error {
	home, err := somUserHome()
	if err != nil {
		return err
	}

	iconDir := filepath.Join(
		home,
		".local",
		"share",
		"icons",
		"hicolor",
		"scalable",
		"apps",
	)

	appDir := filepath.Join(
		home,
		".local",
		"share",
		"applications",
	)

	if err := os.MkdirAll(iconDir, 0o755); err != nil {
		return fmt.Errorf("create icon directory: %w", err)
	}

	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return fmt.Errorf("create application directory: %w", err)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("determine executable path: %w", err)
	}

	desktop := bytes.ReplaceAll(
		somDesktop,
		[]byte("@SOM_EXEC@"),
		[]byte(exe),
	)

	iconPath := filepath.Join(iconDir, "som.svg")
	desktopPath := filepath.Join(appDir, "som.desktop")

	if err := os.WriteFile(iconPath, somIcon, 0o644); err != nil {
		return fmt.Errorf("write icon: %w", err)
	}

	if err := os.WriteFile(desktopPath, desktop, 0o644); err != nil {
		return fmt.Errorf("write desktop entry: %w", err)
	}

	fmt.Println("Installed desktop assets:")
	fmt.Println(" ", iconPath)
	fmt.Println(" ", desktopPath)

	return nil
}

func somUserHome() (string, error) {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		u, err := user.Lookup(sudoUser)
		if err != nil {
			return "", fmt.Errorf("lookup SUDO_USER: %w", err)
		}

		return u.HomeDir, nil
	}

	return os.UserHomeDir()
}
