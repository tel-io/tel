package log

import (
	"context"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type counter struct {
	resetAt int64
	counter uint64
}

type counters [numLevels][4096]counter

func newCounters() *counters {
	return &counters{}
}

func (cs *counters) get(lvl Level, key string) *counter {
	i := IndexLevel(lvl)
	j := fnv32a(key) % 4096

	return &cs[i][j]
}

// fnv32a, adapted from "hash/fnv", but without a []byte(string) alloc.
func fnv32a(str string) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	hash := uint32(offset32)
	for i := 0; i < len(str); i++ {
		hash ^= uint32(str[i])
		hash *= prime32
	}

	return hash
}

func (c *counter) IncCheckReset(t time.Time, tick time.Duration) uint64 {
	tun := t.UnixNano()
	resetAfter := atomic.LoadInt64(&c.resetAt)
	if resetAfter > tun {
		return atomic.AddUint64(&c.counter, 1)
	}

	atomic.StoreUint64(&c.counter, 1)

	newResetAfter := tun + tick.Nanoseconds()
	if !atomic.CompareAndSwapInt64(&c.resetAt, resetAfter, newResetAfter) {
		// We raced with another goroutine trying to reset, and it also reset
		// the counter to 1, so we need to reincrement the counter.
		return atomic.AddUint64(&c.counter, 1)
	}

	return 1
}

type SamplerOption func(*sampler)

func WithSamplerLevelThreshold(level Level, n uint64) SamplerOption {
	return func(s *sampler) {
		s.levelThreshold[IndexLevel(level)] = n
	}
}

func WithSamplerLevelThresholdString(str string) SamplerOption {
	return func(smpl *sampler) {
		pairs := strings.Split(str, ",")
		for _, kvstr := range pairs {
			kval := strings.Split(strings.TrimSpace(kvstr), "=")
			if len(kval) != 2 {
				continue
			}

			level := UnmarshalTextLevel(kval[0])

			threshold, err := strconv.Atoi(kval[1])
			if err != nil {
				continue
			}

			WithSamplerLevelThreshold(level, uint64(threshold))(smpl)
		}
	}
}

func NewSampler(handler Handler, tick time.Duration, threshold, thereafter int, options ...SamplerOption) Handler {
	if threshold <= 0 {
		return handler
	}

	levelThreshold := make([]uint64, numLevels)
	for i := range levelThreshold {
		levelThreshold[i] = uint64(threshold)
	}

	smpl := &sampler{
		h:              handler,
		tick:           tick,
		thereafter:     uint64(thereafter),
		counts:         newCounters(),
		levelThreshold: levelThreshold,
		levelStatus:    make([]uint32, numLevels),
	}

	for _, opt := range options {
		opt(smpl)
	}

	return smpl
}

type sampler struct {
	h Handler

	counts         *counters
	tick           time.Duration
	thereafter     uint64
	levelThreshold []uint64
	levelStatus    []uint32
}

func (s *sampler) Enabled(ctx context.Context, level Level) bool {
	return s.h.Enabled(ctx, level)
}

func (s *sampler) Handle(ctx context.Context, rec Record) error {
	if !s.h.Enabled(ctx, rec.Level) {
		return nil
	}

	threshold := s.levelThreshold[IndexLevel(rec.Level)]
	if threshold <= 0 {
		return s.h.Handle(ctx, rec)
	}

	counter := s.counts.get(rec.Level, rec.Message)
	n := counter.IncCheckReset(rec.Time, s.tick)
	if n == 1 {
		atomic.StoreUint32(&s.levelStatus[IndexLevel(rec.Level)], 0)
	}

	if n > threshold && (s.thereafter == 0 || (n-threshold)%s.thereafter != 0) {
		if !atomic.CompareAndSwapUint32(&s.levelStatus[IndexLevel(rec.Level)], 0, 1) {
			return nil
		}

		rec.Message = "log sampler: threshold has been exceeded"
		rec.Level = LevelWarn
		rec.AddAttrs(
			String("rec_level", StringLevel(rec.Level)),
			Uint64("threshold", threshold),
		)
	}

	return s.h.Handle(ctx, rec)
}

func (s *sampler) WithAttrs(attrs []Attr) Handler {
	cloned := *s
	cloned.h = s.h.WithAttrs(attrs)

	return &cloned
}

func (s *sampler) WithGroup(name string) Handler {
	cloned := *s
	cloned.h = s.h.WithGroup(name)

	return &cloned
}
