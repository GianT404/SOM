package audio

import (
	"sync"
	"time"
)

// PCMSource là nguồn cấp PCM cho Capture — khớp với player.Player.
type PCMSource interface {
	Subscribe() chan []byte
	Unsubscribe(chan []byte)
	BufferedBytes() int
}

const (
	pcmChannels       = 2
	pcmBytesPerSample = 2 // s16le
	pcmSampleRate     = 48000
	pcmBytesPerSecond = pcmSampleRate * pcmChannels * pcmBytesPerSample
)

type bandsSnapshot struct {
	readyAt time.Time
	bands   []float64
}

type Capture struct {
	mu    sync.Mutex
	bands []float64
	stop  chan struct{}
}

func New() *Capture {
	return &Capture{}
}

func (c *Capture) Start(src PCMSource, bands int) error {
	c.mu.Lock()
	if c.stop != nil {
		c.mu.Unlock()
		return nil
	}
	stop := make(chan struct{})
	c.stop = stop
	c.bands = nil
	c.mu.Unlock()

	sub := src.Subscribe()

	go func() {
		defer src.Unsubscribe(sub)

		const frameBytes = pcmChannels * pcmBytesPerSample
		var leftover []byte
		prev := make([]float64, bands)
		var queue []bandsSnapshot

		publishReady := func() {
			now := time.Now()
			for len(queue) > 0 && !now.Before(queue[0].readyAt) {
				c.mu.Lock()
				c.bands = queue[0].bands
				c.mu.Unlock()
				queue = queue[1:]
			}
		}

		for {
			select {
			case <-stop:
				return
			case chunk, ok := <-sub:
				if !ok {
					return
				}

				leftover = append(leftover, chunk...)
				usable := len(leftover) - (len(leftover) % frameBytes)
				if usable <= 0 {
					publishReady()
					continue
				}
				frame := leftover[:usable]
				leftover = append([]byte(nil), leftover[usable:]...)

				n := usable / frameBytes
				samples := make([]float64, n)
				for i := 0; i < n; i++ {
					off := i * frameBytes
					// Stereo interleaved: L(int16) R(int16). Average cả 2 channels.
					l := int16(uint16(frame[off]) | uint16(frame[off+1])<<8)
					r := int16(uint16(frame[off+2]) | uint16(frame[off+3])<<8)
					samples[i] = (float64(l) + float64(r)) / 2.0 / 32768.0
				}

				snap := magnitudeBands(samples, bands)
				for i := range prev {
					if i < len(snap) {
						prev[i] = prev[i]*0.40 + snap[i]*0.60
					}
				}
				smoothed := make([]float64, len(prev))
				copy(smoothed, prev)

				// Delay = đúng bằng thời lượng audio đang xếp hàng chờ trong buffer
				delaySec := float64(src.BufferedBytes()) / float64(pcmBytesPerSecond)
				readyAt := time.Now().Add(time.Duration(delaySec * float64(time.Second)))

				queue = append(queue, bandsSnapshot{readyAt: readyAt, bands: smoothed})
				publishReady()
			}
		}
	}()

	return nil
}

func (c *Capture) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stop != nil {
		close(c.stop)
		c.stop = nil
	}
	c.bands = nil
}

func (c *Capture) Bands() []float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bands
}
