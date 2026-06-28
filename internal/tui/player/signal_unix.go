//go:build !windows

package player

import (
	"os"
	"syscall"
)

func sendSignalPause(proc *os.Process) error  { return proc.Signal(syscall.SIGSTOP) }
func sendSignalResume(proc *os.Process) error { return proc.Signal(syscall.SIGCONT) }
