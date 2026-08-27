package audio

import "sync"

type Capture struct {
	mu    sync.Mutex
	bands []float64
	stop  chan struct{}
}

func New() *Capture {
	return &Capture{}
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
