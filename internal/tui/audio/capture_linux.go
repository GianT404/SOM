//go:build linux

package audio

import (
	"encoding/binary"
	"io"
	"os/exec"
)

const sampleRate = 44100

func platformCapture(bands int, stop chan struct{}, onBands func([]float64)) error {
	cmd := exec.Command("parec",
		"-d", "@DEFAULT_MONITOR@",
		"--rate=44100",
		"--channels=1",
		"--format=s16le",
		"--latency-msec=30",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		<-stop
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	go func() {
		const frameSamples = sampleRate / 30
		raw := make([]byte, frameSamples*2)

		for {
			select {
			case <-stop:
				return
			default:
			}

			if _, err := io.ReadFull(stdout, raw); err != nil {
				return
			}

			samples := make([]float64, frameSamples)
			for i := 0; i < frameSamples; i++ {
				v := int16(binary.LittleEndian.Uint16(raw[2*i : 2*i+2]))
				samples[i] = float64(v) / 32768.0
			}

			onBands(magnitudeBands(samples, bands))
		}
	}()

	return nil
}
