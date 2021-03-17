package syslog

import (
	"sync"
)

// Queue ...
type Queue struct {
	buf            []map[string]interface{}
	flushThreshold int
	maxItems       int
	mu             sync.RWMutex
}

// NewQueue ...
func NewQueue(flushThreshold int) *Queue {
	return NewQueueWithMax(flushThreshold, 0)
}

// NewQueueWithMax ...
func NewQueueWithMax(flushThreshold int, maxItems int) *Queue {
	return &Queue{
		buf:            []map[string]interface{}{},
		flushThreshold: flushThreshold,
		maxItems:       maxItems,
	}
}

// Push will append given messages to the queue, returns number of items
// in the queue after the append, and if the queue was created using
// NewQueueWithMax, number of dropped messages
func (q *Queue) Push(msgs []map[string]interface{}) (int, int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	dropped := 0

	if q.maxItems > 0 && len(q.buf)+len(msgs) > q.maxItems {
		maxIdx := q.maxItems - len(q.buf)
		dropped = len(msgs) - maxIdx
		msgs = msgs[:maxIdx]
	}
	q.buf = append(q.buf, msgs...)

	return len(q.buf), dropped
}

// Get ...
func (q *Queue) Get() []map[string]interface{} {
	return q.GetN(q.flushThreshold)
}

// GetN ...
func (q *Queue) GetN(n int) []map[string]interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()

	min := n
	fetchAll := len(q.buf) <= n
	if len(q.buf) < min {
		min = len(q.buf)
		if min == 0 {
			return []map[string]interface{}{}
		}
	}
	ret := q.buf[:min]
	if fetchAll {
		// this should be cheaper than constantly shifting the view
		q.buf = make([]map[string]interface{}, 0, n)
	} else {
		// FIXME: under heavy load code-path, should probably optimize it somehow
		q.buf = q.buf[min:]
	}

	return ret
}

func (q *Queue) size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.buf)
}
