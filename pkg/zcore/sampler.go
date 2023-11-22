package zcore

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	_maxLevel         = zap.FatalLevel
	_minLevel         = zap.DebugLevel
	_numLevels        = _maxLevel - _minLevel + 1
	_countersPerLevel = 4096
)

func indexLevel(lvl zapcore.Level) int {
	return int(lvl - _minLevel)
}

type counter struct {
	resetAt atomic.Int64
	counter atomic.Uint64
}

type counters [_numLevels][_countersPerLevel]counter

func newCounters() *counters {
	return &counters{}
}

func (cs *counters) get(lvl zapcore.Level, key string) *counter {
	i := indexLevel(lvl)
	j := fnv32a(key) % _countersPerLevel
	return &cs[i][j]
}

// fnv32a, adapted from "hash/fnv", but without a []byte(string) alloc
func fnv32a(s string) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	hash := uint32(offset32)
	for i := 0; i < len(s); i++ {
		hash ^= uint32(s[i])
		hash *= prime32
	}
	return hash
}

func (c *counter) IncCheckReset(t time.Time, tick time.Duration) uint64 {
	tn := t.UnixNano()
	resetAfter := c.resetAt.Load()
	if resetAfter > tn {
		return c.counter.Inc()
	}

	c.counter.Store(1)

	newResetAfter := tn + tick.Nanoseconds()
	if !c.resetAt.CAS(resetAfter, newResetAfter) {
		// We raced with another goroutine trying to reset, and it also reset
		// the counter to 1, so we need to reincrement the counter.
		return c.counter.Inc()
	}

	return 1
}

type SamplerOption func(*sampler)

func WithSamplerLevelThreshold(level zapcore.Level, n uint64) SamplerOption {
	return func(s *sampler) {
		s.levelThreshold[indexLevel(level)] = n
	}
}

func WithSamplerLevelThresholdString(str string) SamplerOption {
	return func(s *sampler) {
		pairs := strings.Split(str, ",")
		for _, kvstr := range pairs {
			kv := strings.Split(strings.TrimSpace(kvstr), "=")
			if len(kv) != 2 {
				continue
			}

			var level zapcore.Level
			if err := level.UnmarshalText([]byte(kv[0])); err != nil {
				continue
			}

			threshold, err := strconv.Atoi(kv[1])
			if err != nil {
				continue
			}

			WithSamplerLevelThreshold(level, uint64(threshold))(s)
		}
	}
}

// NewSampler creates a Core that samples incoming entries, which
// caps the CPU and I/O load of logging while attempting to preserve a
// representative subset of your logs.
//
// Zap samples by logging the first N entries with a given level and message
// each tick. If more Entries with the same level and message are seen during
// the same interval, every Mth message is logged and the rest are dropped.
//
// Sampler can be configured to report sampling decisions with the SamplerHook
// option.
//
// Keep in mind that zap's sampling implementation is optimized for speed over
// absolute precision; under load, each tick may be slightly over- or
// under-sampled.
func NewSampler(
	core zapcore.Core,
	tick time.Duration,
	threshold, thereafter int,
	options ...SamplerOption,
) zapcore.Core {
	levelThreshold := make([]uint64, _numLevels)
	for i := range levelThreshold {
		levelThreshold[i] = uint64(threshold)
	}

	s := &sampler{
		Core:           core,
		tick:           tick,
		counts:         newCounters(),
		thereafter:     uint64(thereafter),
		levelThreshold: levelThreshold,
		levelStatus:    make([]atomic.Uint32, _numLevels),
	}

	for _, opt := range options {
		opt(s)
	}

	return s
}

type sampler struct {
	zapcore.Core

	counts         *counters
	tick           time.Duration
	thereafter     uint64
	levelThreshold []uint64
	levelStatus    []atomic.Uint32
}

func (s *sampler) With(fields []zapcore.Field) zapcore.Core {
	return &sampler{
		Core:           s.Core.With(fields),
		tick:           s.tick,
		counts:         s.counts,
		thereafter:     s.thereafter,
		levelThreshold: s.levelThreshold,
		levelStatus:    s.levelStatus,
	}
}

func (s *sampler) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !s.Enabled(ent.Level) {
		return ce
	}

	levelIdx := indexLevel(ent.Level)
	threshold := s.levelThreshold[levelIdx]
	if threshold <= 0 {
		return s.Core.Check(ent, ce)
	}

	counter := s.counts.get(ent.Level, ent.Message)
	n := counter.IncCheckReset(ent.Time, s.tick)
	if n == 1 {
		s.levelStatus[levelIdx].Store(0)
	}

	if n > threshold && (s.thereafter == 0 || (n-threshold)%s.thereafter != 0) {
		if !s.levelStatus[levelIdx].CAS(0, 1) {
			return ce
		}

		msg := fmt.Sprintf("log sampler: threshold has been exceeded. level: %s, threshold: %d/sec", ent.Level, threshold)
		ent.Message = msg
		if ce != nil {
			ce.Entry.Message = msg
		}
	}

	return s.Core.Check(ent, ce)
}
