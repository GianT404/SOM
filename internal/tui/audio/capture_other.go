//go:build !linux

package audio

import "fmt"

type unsupportedCapture struct{}

func newPlatformCapture() platformCapture { return &unsupportedCapture{} }

func (u *unsupportedCapture) probe() error {
	return fmt.Errorf("audio visualizer capture is not implemented on this OS yet")
}

func (u *unsupportedCapture) run(bands int, out chan<- []float64, stop <-chan struct{}) error {
	return fmt.Errorf("not supported")
}
