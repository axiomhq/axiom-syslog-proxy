package input

import (
	"io"
	"sync"
)

// NotifyCloser ...
type NotifyCloser struct {
	toClose   io.Closer
	close     chan struct{}
	mutex     sync.Mutex
	wasClosed bool
}

// NewNotifyCloser ...
func NewNotifyCloser(toClose io.Closer) *NotifyCloser {
	return &NotifyCloser{
		toClose: toClose,
		close:   make(chan struct{}, 1),
	}
}

// Close ...
func (c *NotifyCloser) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.wasClosed = true
	select {
	case c.close <- struct{}{}:
	default:
	}

	return c.toClose.Close()
}

// CloseChan ...
func (c *NotifyCloser) CloseChan() chan struct{} {
	return c.close
}

// WasClosed returns true if this Closable was closed
func (c *NotifyCloser) WasClosed() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.wasClosed
}
