package log

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAttr(t *testing.T) {
	assert := assert.New(t)

	for _, test := range []struct {
		attr        func() Attr
		expectedKey string
	}{
		{
			attr:        func() Attr { return AttrTime(time.Now()) },
			expectedKey: AttrKeyTime,
		},
		{
			attr:        func() Attr { return AttrSpanID("span_id") },
			expectedKey: AttrKeySpanID,
		},
		{
			attr:        func() Attr { return AttrTraceFlags(1) },
			expectedKey: AttrKeyTraceFlags,
		},
		{
			attr:        func() Attr { return AttrSource("source") },
			expectedKey: AttrKeySource,
		},
		{
			attr:        func() Attr { return AttrMsg("msg") },
			expectedKey: AttrKeyMsg,
		},
		{
			attr:        func() Attr { return AttrLevel("level") },
			expectedKey: AttrKeyLevel,
		},
		{
			attr:        func() Attr { return AttrStack("stack") },
			expectedKey: AttrKeyStack,
		},
		{
			attr:        func() Attr { return AttrCallerSkipOffset(1) },
			expectedKey: AttrKeyCallerSkipOffset,
		},
	} {
		assert.Equal(test.expectedKey, test.attr().Key)
	}
}

func TestAttr_pairs(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(
		[]Attr{String("count", "1"), String("total_dropped", "0")},
		pairs([]interface{}{"count", 1, "total_dropped", 0}),
	)
}
