package log

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

var _ Handler = (*teeHandler)(nil)

//nolint:gochecknoglobals
var (
	IgnorePC  bool
	AddSource = true
)

func writeThenActionByLevel(level Level) WriteThenAction {
	switch level { //nolint:exhaustive
	case LevelPanic:
		return WriteThenPanic
	case LevelFatal:
		return WriteThenFatal
	default:
		return WriteThenNoop
	}
}

func doWriteThenAction(action WriteThenAction, msg string) {
	switch action { //nolint:exhaustive
	case WriteThenPanic:
		panic(msg)
	case WriteThenFatal:
		os.Exit(1)
	}
}

func callerProgramCounter(pc uintptr, skipOffset int) uintptr {
	if !IgnorePC && pc == 0 {
		var pcs [1]uintptr
		runtime.Callers(4+skipOffset, pcs[:])
		pc = pcs[0]
	}

	return pc
}

func NewTeeHandler(handler Handler, other ...Handler) Handler {
	return &teeHandler{
		h:          handler,
		hs:         other,
		attrIDs:    make(map[string]int),
		skipOffset: 2,
		mu:         new(sync.Mutex),
	}
}

type teeHandler struct {
	h         Handler
	hs        []Handler
	attrIDs   map[string]int
	attrs     []Attr
	spanAttrs []Attr

	name       []byte
	skipOffset int
	pc         uintptr

	mu *sync.Mutex
}

func (h *teeHandler) cloneAttrs(n int) *teeHandler {
	cloned := *h
	cloned.mu = new(sync.Mutex)

	h.mu.Lock()
	cloned.attrIDs = make(map[string]int, len(h.attrIDs)+n)
	for k, v := range h.attrIDs {
		cloned.attrIDs[k] = v
	}

	if len(h.attrs) > 0 {
		cloned.attrs = make([]Attr, len(h.attrs))
		copy(cloned.attrs, h.attrs)
	}

	if len(h.spanAttrs) > 0 {
		cloned.spanAttrs = make([]Attr, len(h.spanAttrs))
		copy(cloned.spanAttrs, h.spanAttrs)
	}
	h.mu.Unlock()

	return &cloned
}

func (h *teeHandler) cloneName() *teeHandler {
	cloned := *h
	cloned.mu = new(sync.Mutex)

	h.mu.Lock()
	if len(h.name) > 0 {
		cloned.name = make([]byte, len(h.name))
		copy(cloned.name, h.name)
	}
	h.mu.Unlock()

	return &cloned
}

func (h *teeHandler) newRecord(rec Record) (Record, WriteThenAction) {
	writeThenAction := writeThenActionByLevel(rec.Level)
	attrh := &attrHandler{
		attrs:            make([]Attr, len(h.attrs), len(h.attrs)+len(h.spanAttrs)+rec.NumAttrs()+1),
		attrIDs:          h.attrIDs,
		spanAttrs:        h.spanAttrs,
		callerSkipOffset: h.skipOffset,
		callerPC:         h.pc,
	}
	copy(attrh.attrs, h.attrs)

	rec.Attrs(func(attr Attr) bool {
		attrh.handle(attr)

		return true
	})

	attrs := append(attrh.attrs, attrh.spanAttrs...) //nolint: gocritic
	if len(h.name) > 0 {
		attrs = append(attrs, AttrLoggerName(string(h.name)))
	}

	pc := callerProgramCounter(attrh.callerPC, attrh.callerSkipOffset)
	rec = NewRecord(time.Now(), rec.Level, rec.Message, pc)
	rec.AddAttrs(attrs...)

	return rec, writeThenAction
}

func (h *teeHandler) Enabled(ctx context.Context, level Level) bool {
	enabled := h.h.Enabled(ctx, level)
	if enabled {
		return true
	}

	for _, handler := range h.hs {
		enabled = handler.Enabled(ctx, level) || enabled
		if enabled {
			return true
		}
	}

	return false
}

func (h *teeHandler) Handle(ctx context.Context, rec Record) error {
	rec, writeThenAction := h.newRecord(rec)

	var herr error
	if h.h.Enabled(ctx, rec.Level) {
		herr = h.h.Handle(ctx, rec)
	}

	for _, handler := range h.hs {
		if handler.Enabled(ctx, rec.Level) {
			if err := handler.Handle(ctx, rec); err != nil && herr == nil {
				herr = err
			}
		}
	}

	doWriteThenAction(writeThenAction, rec.Message)

	return herr
}

func (h *teeHandler) WithAttrs(attrs []Attr) Handler {
	if len(attrs) == 0 {
		return h
	}

	cloned := h.cloneAttrs(len(attrs))
	attrh := &attrHandler{
		attrs:            cloned.attrs,
		attrIDs:          cloned.attrIDs,
		spanAttrs:        cloned.spanAttrs,
		callerSkipOffset: cloned.skipOffset,
		callerPC:         cloned.pc,
		indexable:        true,
	}

	for _, attr := range attrs {
		attrh.handle(attr)
	}

	cloned.attrs = attrh.attrs
	cloned.attrIDs = attrh.attrIDs
	cloned.spanAttrs = attrh.spanAttrs
	cloned.skipOffset = attrh.callerSkipOffset
	cloned.pc = attrh.callerPC

	return cloned
}

func (h *teeHandler) WithGroup(name string) Handler {
	if len(name) == 0 {
		return h
	}

	cloned := h.cloneName()

	if len(cloned.name) > 0 {
		cloned.name = append(cloned.name, '.')
	}
	cloned.name = append(cloned.name, []byte(name)...)

	return cloned
}

type attrHandler struct {
	attrs            []Attr
	attrIDs          map[string]int
	spanAttrs        []Attr
	callerSkipOffset int
	callerPC         uintptr
	indexable        bool
}

func (h *attrHandler) handle(attr Attr) { //nolint:funlen, cyclop
	if attr.Key == "" || attr.Equal(EmptyAttr) {
		return
	}

	var attrStack *Attr
	switch attr.Key {
	case AttrKeySpan:
		anyValue := attr.Value.Any()
		if anyValue == nil {
			h.spanAttrs = nil

			return
		}

		span := anyValue.(trace.Span) //nolint:forcetypeassert
		spanAttrs := make([]Attr, 0, 3)

		traceIDAttr := AttrTraceID(span.SpanContext().TraceID().String())
		if h.exists(AttrKeyTraceID) {
			h.appdate(traceIDAttr)
		} else {
			spanAttrs = append(spanAttrs, traceIDAttr)
		}

		spanIDAttr := AttrSpanID(span.SpanContext().SpanID().String())
		if h.exists(AttrKeySpanID) {
			h.appdate(spanIDAttr)
		} else {
			spanAttrs = append(spanAttrs, spanIDAttr)
		}

		traceFlagsAttr := AttrTraceFlags(int(span.SpanContext().TraceFlags()))
		if h.exists(AttrKeyTraceFlags) {
			h.appdate(traceFlagsAttr)
		} else {
			spanAttrs = append(spanAttrs, traceFlagsAttr)
		}

		h.spanAttrs = spanAttrs

		return
	case AttrKeyCallerSkipOffset:
		h.callerSkipOffset += int(attr.Value.Int64())

		return
	case AttrKeyCallerPC:
		h.callerPC = uintptr(attr.Value.Int64())

		return
	case AttrKeyError:
		value := attr.Value.Any()
		switch t := value.(type) {
		case *errorNamed:
			attr = String(t.name, t.Error())
			if stack := Stack(t); stack.Key != "" {
				attrStack = &stack
			}
		case LogValuer:
			attr = Any(AttrKeyError, t.LogValue())
		case error:
			attr = String(AttrKeyError, t.Error())
			if stack := Stack(t); stack.Key != "" {
				attrStack = &stack
			}
		case string:
			attr = String(AttrKeyError, t)
		default:
			return
		}
	}

	h.appdate(attr)
	h.appdatePtr(attrStack)
}

func (h *attrHandler) exists(attrKey string) bool {
	_, ok := h.attrIDs[attrKey]

	return ok
}

func (h *attrHandler) appdate(attr Attr) {
	if idx, ok := h.attrIDs[attr.Key]; ok {
		h.attrs[idx] = attr
	} else {
		h.attrs = append(h.attrs, attr)
		if h.indexable {
			h.attrIDs[attr.Key] = len(h.attrs) - 1
		}
	}
}

func (h *attrHandler) appdatePtr(attr *Attr) {
	if attr == nil {
		return
	}

	h.appdate(*attr)
}
