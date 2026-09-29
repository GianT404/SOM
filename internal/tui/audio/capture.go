package audio

import "sync"

// PCMSource supplies the exact PCM stream being played by SOM.
type PCMSource interface {
	Subscribe() chan []byte
	Unsubscribe(chan []byte)
}

const (
	pcmChannels       = 2
	pcmBytesPerSample = 2
)

type Capture struct {
	mu    sync.Mutex
	bands []float64
	stop  chan struct{}
}

func New() *Capture {
	return &Capture{}
}

func (c *Capture) StartPCM(src PCMSource, bands int) error {
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
					left := int16(uint16(frame[off]) | uint16(frame[off+1])<<8)
					right := int16(uint16(frame[off+2]) | uint16(frame[off+3])<<8)
					samples[i] = (float64(left) + float64(right)) / 2.0 / 32768.0
				}
				bandsOut := magnitudeBands(samples, bands)
				c.mu.Lock()
				c.bands = bandsOut
				c.mu.Unlock()
			}
		}
	}()
	return nil
}

func (c *Capture) Start(bands int) error {
	c.mu.Lock()
	if c.stop != nil {
		c.mu.Unlock()
		return nil
	}
	stop := make(chan struct{})
	c.stop = stop
	c.bands = nil
	c.mu.Unlock()

	return platformCapture(bands, stop, func(b []float64) {
		c.mu.Lock()
		c.bands = b
		c.mu.Unlock()
	})
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
