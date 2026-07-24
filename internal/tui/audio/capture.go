package audio

import "sync"

type platformCapture interface {
	probe() error
	run(bands int, out chan<- []float64, stop <-chan struct{}) error
}

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

	impl := newPlatformCapture()
	if err := impl.probe(); err != nil {
		c.mu.Unlock()
		return err
	}

	stop := make(chan struct{})
	c.stop = stop
	c.bands = nil
	c.mu.Unlock()

	out := make(chan []float64, 2)

	go func() {
		_ = impl.run(bands, out, stop)
	}()

	go func() {
		prev := make([]float64, bands)
		for {
			select {
			case <-stop:
				return
			case snap, ok := <-out:
				if !ok {
					return
				}
				for i := range prev {
					if i < len(snap) {
						prev[i] = prev[i]*0.55 + snap[i]*0.45
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
