//go:build !linux && !windows

package audio

import "errors"

// platformCapture chưa hỗ trợ trên OS này
func platformCapture(bands int, stop chan struct{}, onBands func([]float64)) error {
	return errors.New("audio loopback capture: not support")
}
