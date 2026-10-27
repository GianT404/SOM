package fileutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// WriteUnique writes one complete file to a temporary path in dir, then
// publishes it at a name that does not already exist. The temporary file is
// on the same filesystem as the destination so the hard-link publication is
// atomic on filesystems that support links.
func WriteUnique(dir, baseName, discriminator, ext string, write func(io.Writer) error) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	baseName = cleanComponent(baseName)
	if baseName == "" {
		baseName = "track"
	}
	discriminator = cleanComponent(discriminator)

	tmp, err := os.CreateTemp(dir, ".som-download-*.tmp")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := write(tmp); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}

	for n := 0; ; n++ {
		name := uniqueName(baseName, discriminator, ext, n)
		dest := filepath.Join(dir, name)
		linkErr := os.Link(tmpPath, dest)
		if linkErr == nil {
			return dest, nil
		} else if errors.Is(linkErr, os.ErrExist) {
			continue
		} else {
			// Some filesystems do not support hard links. O_EXCL still makes
			// destination creation race-safe and prevents replacing an existing file.
			if err := copyToExclusive(tmpPath, dest); err == nil {
				return dest, nil
			} else if errors.Is(err, os.ErrExist) {
				continue
			} else {
				return "", fmt.Errorf("publish download: %w (hard link: %v)", err, linkErr)
			}
		}
	}
}

func uniqueName(base, discriminator, ext string, n int) string {
	switch {
	case n == 0:
		return base + ext
	case n == 1 && discriminator != "":
		return base + " (" + discriminator + ")" + ext
	case discriminator != "":
		return base + " (" + discriminator + " " + strconv.Itoa(n) + ")" + ext
	default:
		return base + " (" + strconv.Itoa(n) + ")" + ext
	}
}

func cleanComponent(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' || r < 0x20 {
			return '-'
		}
		return r
	}, strings.TrimSpace(s))
}

func copyToExclusive(src, dest string) error {
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}

	in, err := os.Open(src)
	if err == nil {
		_, err = io.Copy(out, in)
		_ = in.Close()
	}
	if err == nil {
		err = out.Sync()
	}
	closeErr := out.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(dest)
		return err
	}
	return nil
}
