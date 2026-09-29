package intent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultModelPath     = "~/.local/share/som/voice/intent.json"
	SystemModelPath      = "/usr/local/share/som/voice/intent.json"
	RepoModelPath        = "internal/voice/intent/model.json"
	DefaultMinConfidence = 0.30
)

func ResolveModelPath(configured string) (string, error) {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		configured = DefaultModelPath
	}

	candidates := []string{expandUserPath(configured)}
	if configured == DefaultModelPath {
		candidates = append(candidates, SystemModelPath, RepoModelPath)
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil {
			if info.IsDir() {
				return "", fmt.Errorf("intent model path is a directory: %s", candidate)
			}
			return candidate, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("stat intent model %s: %w", candidate, err)
		}
	}

	return "", fmt.Errorf("intent model not found (checked %s)", strings.Join(candidates, ", "))
}

func expandUserPath(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
		return path
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
