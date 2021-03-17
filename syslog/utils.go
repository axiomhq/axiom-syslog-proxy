package syslog

import (
	"sync/atomic"
	"time"

	"axicode.axiom.co/watchmakers/watchly/pkg/datastores/settings"
)

// ShardQueryOptimizationsDisableKey ...
type ShardQueryOptimizationsDisableKey struct{}

var (
	// All this shit belongs in a config
	maxNumDocs         uint64
	maxBatchSize       uint64
	cacheSize          uint64
	flushTick          int64
	flushThreshold     uint64
	retentionCheckTick int64

	retentionUnixNano = int64(0)
	initDone          = int64(0)
)

// InitVars ...
func InitVars() {
	if atomic.CompareAndSwapInt64(&initDone, 0, 1) {
		maxNumDocs = settings.GetLogsShardSize()
		maxBatchSize = settings.GetLogsBatchSize()
		cacheSize = settings.GetLogsCacheSize()
		flushTick = int64(settings.GetLogsFlushTick())
		flushThreshold = settings.GetLogsFlushThreshold()
		retentionCheckTick = int64(time.Minute * 5)
	}
}

const (
	// NanoSecondsInDay ...
	NanoSecondsInDay = 86400000000000
)

// GetFlushThreshold ...
func GetFlushThreshold() int {
	return int(atomic.LoadUint64(&flushThreshold))
}

// SetFlushThreshold ...
func SetFlushThreshold(n int) {
	atomic.StoreUint64(&flushThreshold, uint64(n))
}

// GetCacheSize ...
func GetCacheSize() int {
	return int(atomic.LoadUint64(&cacheSize))
}

// SetCacheSize ...
func SetCacheSize(n int) {
	atomic.StoreUint64(&cacheSize, uint64(n))
}

// GetFlushTick ...
func GetFlushTick() time.Duration {
	return time.Duration(atomic.LoadInt64(&flushTick))
}

// SetFlushTick ...
func SetFlushTick(n time.Duration) {
	atomic.StoreInt64(&flushTick, int64(n))
}

// GetRetentionCheckTick ...
func GetRetentionCheckTick() time.Duration {
	return time.Duration(atomic.LoadInt64(&retentionCheckTick))
}

// SetRetentionCheckTick ...
func SetRetentionCheckTick(n time.Duration) {
	atomic.StoreInt64(&retentionCheckTick, int64(n))
}

// GetMaxNumDocs ...
func GetMaxNumDocs() uint64 {
	return atomic.LoadUint64(&maxNumDocs)
}

// SetMaxNumDocs ...
func SetMaxNumDocs(n uint64) {
	atomic.StoreUint64(&maxNumDocs, n)
}

// GetMaxBatchSize ...
func GetMaxBatchSize() uint64 {
	return atomic.LoadUint64(&maxBatchSize)
}

// SetMaxBatchSize ...
func SetMaxBatchSize(n uint64) {
	atomic.StoreUint64(&maxBatchSize, n)
}

// GetRetention ...
func GetRetention() time.Duration {
	return time.Duration(atomic.LoadInt64(&retentionUnixNano))
}

// SetRetention ...
func SetRetention(r time.Duration) {
	atomic.StoreInt64(&retentionUnixNano, int64(r))

}

// GetRetentionDays ...
func GetRetentionDays() time.Duration {
	settings.GetLogsRetentionDays()
	ret := GetRetention()

	if ret == 0 {
		ret = time.Duration(settings.GetLogsRetentionDays()) * 24 * time.Hour
		if atomic.CompareAndSwapInt64(&retentionUnixNano, 0, int64(ret)) {
			return ret
		}
	}
	return ret
}
