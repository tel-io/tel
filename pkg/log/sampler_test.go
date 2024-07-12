package log

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSampler(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()
	echo := make(chan Record, 1)
	handler := NewSampler(NewEchoHandler(echo), time.Second, 1, 0)

	for i := 0; i < 10; i++ {
		err := handler.Handle(ctx, NewRecord(time.Now(), LevelInfo, "msg", 0))
		assert.NoError(err)

		select {
		case rec := <-echo:
			if i == 1 {
				assert.Equal("log sampler: threshold has been exceeded", rec.Message)
			}
		default:
		}
	}
}

func TestWithSamplerLevelThreshold(t *testing.T) {
	assert := assert.New(t)

	handler := NewSampler(
		nil,
		0,
		1,
		0,
		WithSamplerLevelThreshold(LevelError, 10),
		WithSamplerLevelThreshold(LevelWarn, 100),
		WithSamplerLevelThreshold(LevelError, 100),
	)
	sampler := handler.(*sampler)

	assert.Len(sampler.levelThreshold, numLevels)
	assert.Equal(uint64(100), sampler.levelThreshold[IndexLevel(LevelWarn)])
	assert.Equal(uint64(100), sampler.levelThreshold[IndexLevel(LevelError)])
}

func TestWithSamplerLevelThresholdString(t *testing.T) {
	assert := assert.New(t)

	for _, tt := range []struct {
		input    string
		expected []uint64
	}{
		{
			input:    "",
			expected: []uint64{0, 0, 0, 0, 0, 0, 0},
		},
		{
			input:    "error=10",
			expected: []uint64{0, 0, 0, 0, 10, 0, 0},
		},
		{
			input:    "trace=1,info=0,warn=10,error=100",
			expected: []uint64{1, 0, 0, 10, 100, 0, 0},
		},
	} {
		sampler := &sampler{
			levelThreshold: make([]uint64, numLevels),
		}

		WithSamplerLevelThresholdString(tt.input)(sampler)
		assert.Equal(tt.expected, sampler.levelThreshold)
	}
}
