package audio

import "sync"

type PCMSource interface {
	Subscribe() chan []byte
	Unsubscribe(chan []byte)
}

const pcmChannels = 2

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

		const frameBytes = pcmChannels * 2
		var leftover []byte
		prev := make([]float64, bands)

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
					continue
				}
				frame := leftover[:usable]
				leftover = append([]byte(nil), leftover[usable:]...)

				n := usable / frameBytes
				samples := make([]float64, n)
				for i := 0; i < n; i++ {
					off := i * frameBytes
					v := int16(uint16(frame[off]) | uint16(frame[off+1])<<8)
					samples[i] = float64(v) / 32768.0
				}

				snap := magnitudeBands(samples, bands)
				for i := range prev {
					if i < len(snap) {
						prev[i] = prev[i]*0.35 + snap[i]*0.75
					}
				}
				smoothed := make([]float64, len(prev))
				copy(smoothed, prev)

				c.mu.Lock()
				c.bands = smoothed
				c.mu.Unlock()
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
