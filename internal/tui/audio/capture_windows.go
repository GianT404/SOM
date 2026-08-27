//go:build windows

package audio

import (
	"github.com/gen2brain/malgo"
)

func platformCapture(
	bands int,
	stop chan struct{},
	onBands func([]float64),
) error {
	ctx, err := malgo.InitContext(
		nil,
		malgo.ContextConfig{},
		nil,
	)
	if err != nil {
		return err
	}

	const sampleRate = 44100

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Loopback)

	deviceConfig.SampleRate = sampleRate
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = 1

	const frameBytes = 2 // S16 mono = 2 bytes/sample

	var leftover []byte

	callbacks := malgo.DeviceCallbacks{
		Data: func(_, input []byte, _ uint32) {
			if len(input) == 0 {
				return
			}

			leftover = append(leftover, input...)

			usable := len(leftover) - (len(leftover) % frameBytes)
			if usable == 0 {
				return
			}

			chunk := leftover[:usable]

			leftover = append([]byte(nil), leftover[usable:]...)

			sampleCount := usable / frameBytes
			samples := make([]float64, sampleCount)

			for i := 0; i < sampleCount; i++ {
				offset := i * frameBytes

				// S16 little-endian.
				raw := uint16(chunk[offset]) |
					uint16(chunk[offset+1])<<8

				sample := int16(raw)
				samples[i] = float64(sample) / 32768.0
			}

			onBands(magnitudeBands(samples, bands))
		},
	}

	device, err := malgo.InitDevice(
		ctx.Context,
		deviceConfig,
		callbacks,
	)
	if err != nil {
		_ = ctx.Uninit()
		ctx.Free()
		return err
	}

	if err := device.Start(); err != nil {
		device.Uninit()
		_ = ctx.Uninit()
		ctx.Free()
		return err
	}

	go func() {
		<-stop

		device.Uninit()
		_ = ctx.Uninit()
		ctx.Free()
	}()

	return nil
}
