//go:build windows

package player

import "os"

func sendSignalPause(proc *os.Process) error  { return nil }
func sendSignalResume(proc *os.Process) error { return nil }
