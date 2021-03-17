package syslog

import (
	"sync/atomic"
	"testing"
	"time"

	"axicode.axiom.co/watchmakers/watchly/pkg/common/testutil"

	"github.com/stretchr/testify/assert"
)

func TestTicker(t *testing.T) {
	ticker := NewTicker(time.Millisecond * 5)

	<-ticker.GetTicker()
	ticker.Stop()
	<-ticker.Done
}

func TestTickerRun(t *testing.T) {
	ticker := NewTicker(time.Millisecond * 5)

	var timesRan uint64
	go func() {
		err := ticker.Run(func() error {
			atomic.AddUint64(&timesRan, 1)
			return nil
		})

		assert.NoError(t, err)
	}()

	// ensure ticker.Run runs
	testutil.Eventually(t, func(t testutil.T) {
		assert.NotZero(t, atomic.LoadUint64(&timesRan))
	})

	ticker.Stop()
	testutil.Eventually(t, func(t testutil.T) {
		select {
		case <-ticker.Done:
		default:
			t.Errorf("ticker.Done does not return")
		}
	})
}
