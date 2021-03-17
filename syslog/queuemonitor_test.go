package syslog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueueMonitor(t *testing.T) {
	assert := assert.New(t)

	qm := newQueueMonitor()
	// don't want the timer to run itself
	qm.Stop()

	qm.swapAndCalcProcessRate()
	assert.Equal(1.0, qm.GetProcessedRate())

	qm.AddQueued(500)
	qm.swapAndCalcProcessRate()
	assert.Equal(0.0, qm.GetProcessedRate())

	qm.AddQueued(500)
	qm.AddProcessed(500)
	qm.swapAndCalcProcessRate()
	assert.Equal(1.0, qm.GetProcessedRate())

	qm.AddQueued(500)
	qm.AddProcessed(250)
	qm.swapAndCalcProcessRate()
	assert.Equal(2.0, qm.GetProcessedRate())

	qm.AddQueued(500)
	qm.AddProcessed(5000)
	qm.swapAndCalcProcessRate()
	assert.Equal(0.1, qm.GetProcessedRate())
}
