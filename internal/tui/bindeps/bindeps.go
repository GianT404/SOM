package bindeps

import (
	"os"
	"path/filepath"
	"runtime"
)

func Find(name string) string {
	exePath, err := os.Executable()
	if err != nil {
		return name
	}

	candidate := name
	if runtime.GOOS == "windows" && filepath.Ext(candidate) == "" {
		candidate += ".exe"
	}

	local := filepath.Join(filepath.Dir(exePath), "bin", candidate)
	if info, statErr := os.Stat(local); statErr == nil && !info.IsDir() {
		return local
	}

	return name
}
