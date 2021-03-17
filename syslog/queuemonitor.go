package syslog

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type queueMonitor struct {
	mu     sync.Mutex
	ticker *Ticker

	queuedLogs    uint64
	processedLogs uint64

	processRate uint64
}

// NewQueueMonitor ...
func newQueueMonitor() *queueMonitor {
	ticker := NewTicker(time.Second * 15)

	m := &queueMonitor{ticker: ticker}
	go m.run()

	return m
}

func (m *queueMonitor) run() {
	m.ticker.Run(func() error {
		m.swapAndCalcProcessRate()
		return nil
	})
}

func (m *queueMonitor) swapAndCalcProcessRate() {
	queued := atomic.SwapUint64(&m.queuedLogs, 0)
	processed := atomic.SwapUint64(&m.processedLogs, 0)

	if queued == 0 {
		atomic.StoreUint64(&m.processRate, math.Float64bits(1))
		return
	}

	if processed == 0 {
		atomic.StoreUint64(&m.processRate, math.Float64bits(0))
		return
	}

	ratio := float64(queued) / float64(processed)
	atomic.StoreUint64(&m.processRate, math.Float64bits(ratio))
}

func (m *queueMonitor) Stop() {
	m.ticker.Stop()
}

func (m *queueMonitor) AddQueued(num uint64) {
	atomic.AddUint64(&m.queuedLogs, num)
}

func (m *queueMonitor) AddProcessed(num uint64) {
	atomic.AddUint64(&m.processedLogs, num)
}

// GetProcessedRate will return the ratio of queued vs processed logs, might return 0
// if no logs were processed, but there were a bunch of logs queued.
// Will return >1.0 if more logs are being queued than processed
func (m *queueMonitor) GetProcessedRate() float64 {
	return math.Float64frombits(atomic.LoadUint64(&m.processRate))
}
