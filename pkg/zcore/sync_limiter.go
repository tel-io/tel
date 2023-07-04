package zcore

import (
	"sync"
	"time"
)

func newSyncLimiter(interval time.Duration) *syncLimiter {
	return &syncLimiter{
		interval: interval,
	}
}

type syncLimiter struct {
	interval     time.Duration
	lastSyncTime time.Time
	mu           sync.Mutex
}

func (lim *syncLimiter) CanSync() bool {
	if lim.interval <= 0 {
		return true
	}

	lim.mu.Lock()
	defer lim.mu.Unlock()

	now := time.Now()
	canSync := now.Sub(lim.lastSyncTime) >= lim.interval
	if canSync {
		lim.lastSyncTime = now
	}

	return canSync
}
