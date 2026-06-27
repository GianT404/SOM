//go:build windows

package player

import "os"

func sendSignalPause(_ *os.Process) error  { return nil }
func sendSignalResume(_ *os.Process) error { return nil }
