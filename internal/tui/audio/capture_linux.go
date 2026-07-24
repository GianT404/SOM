package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
)

const sampleRate = 44100

type linuxCapture struct{}

func newPlatformCapture() platformCapture { return &linuxCapture{} }

func (l *linuxCapture) probe() error {
	if _, err := exec.LookPath("parec"); err != nil {
		return fmt.Errorf("parec not found in PATH (cần pulseaudio-utils hoặc pipewire-pulse)")
	}
	return nil
}

func (l *linuxCapture) run(bands int, out chan<- []float64, stop <-chan struct{}) error {
	cmd := exec.Command("parec",
		"-d", "@DEFAULT_MONITOR@",
		"--format=s16le",
		"--rate=44100",
		"--channels=1",
		"--latency-msec=30",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	go func() {
		<-stop
		_ = cmd.Process.Kill()
	}()

	const frameSamples = sampleRate / 30
	raw := make([]byte, frameSamples*2)

	for {
		select {
		case <-stop:
			return nil
		default:
		}

		if _, err := io.ReadFull(stdout, raw); err != nil {
			return err
		}

		samples := make([]float64, frameSamples)
		for i := 0; i < frameSamples; i++ {
			v := int16(binary.LittleEndian.Uint16(raw[2*i : 2*i+2]))
			samples[i] = float64(v) / 32768.0
		}

		bandsOut := magnitudeBands(samples, bands)

		select {
		case out <- bandsOut:
		case <-stop:
			return nil
		default:
		}
	}
}
