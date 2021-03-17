package syslog

import "time"

// Ticker holds a channel that delivers 'ticks' of a clock at intervals
type Ticker struct {
	ticker *time.Ticker
	Done   chan struct{}
}

// NewTicker returns a new Ticker containing a channel that will send the time with a period specified by the duration argument.
// It adjusts the intervals or drops ticks to make up for slow receivers.
// The duration d must be greater than zero; if not, NewTicker will panic. Stop the ticker to release associated resources.
func NewTicker(d time.Duration) *Ticker {
	ticker := &Ticker{time.NewTicker(d), make(chan struct{}, 1)}
	return ticker
}

// Stop turns off the ticker. After stop no more ticks will be sent and the done channel will be closed
func (t *Ticker) Stop() {
	t.ticker.Stop()
	close(t.Done)
}

// GetTicker will return the ticker channel you can select on, note that the channel will not close after calling Stop()
func (t *Ticker) GetTicker() <-chan time.Time {
	return t.ticker.C
}

// Run will keep invoking the given function on every tick, and return after the function returns a non-nil error
// or Stop() is called
func (t *Ticker) Run(fn func() error) error {
	for {
		select {
		case <-t.Done:
			return nil
		case <-t.ticker.C:
			if fnErr := fn(); fnErr != nil {
				return fnErr
			}
		}
	}
}
