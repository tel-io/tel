package cardinalitydetector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tel-io/tel/v2/pkg/global"
	"github.com/tel-io/tel/v2/pkg/log"
	"go.opentelemetry.io/otel/attribute"
)

var _ Detector = (*detector)(nil)
var _ Detector = (*noopDetector)(nil)

var noopDetectorInstance = &noopDetector{} //nolint:gochecknoglobals

type Detector interface {
	CheckAttrs(context.Context, []attribute.KeyValue) bool
	Shutdown()
}

func New(ctx context.Context, name string, opts Options) Detector {
	if !opts.Enable || opts.MaxCardinality <= 0 {
		return noopDetectorInstance
	}

	if ctx == nil {
		ctx = context.Background()
	}

	detector := &detector{
		opts:            opts,
		name:            name,
		attrs:           make(map[string]map[string]struct{}),
		highCardinality: make(map[string]struct{}),
	}

	if opts.CheckInterval > 0 {
		detector.checkTicker = time.NewTicker(opts.CheckInterval)
		detector.checkDone = make(chan struct{})
		go detector.checkLoop(ctx)
	}

	return detector
}

type detector struct {
	opts            Options
	name            string
	attrs           map[string]map[string]struct{}
	highCardinality map[string]struct{}
	checkTicker     *time.Ticker
	checkDone       chan struct{}

	closed bool

	mu sync.Mutex
}

func (d *detector) checkLoop(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			global.Error(fmt.Errorf("%+v", r), "detector check loop") //nolint:goerr113
		}
	}()

	for {
		select {
		case <-d.checkTicker.C:
			d.checkAllAttrs(ctx)
		case <-ctx.Done():
			return
		case <-d.checkDone:
			return
		}
	}
}

func (d *detector) checkAllAttrs(ctx context.Context) {
	d.mu.Lock()
	attrsLn := len(d.attrs)
	highCardinalityAttrs := make([]string, 0, len(d.highCardinality))
	for attr := range d.highCardinality {
		highCardinalityAttrs = append(highCardinalityAttrs, attr)
	}
	d.mu.Unlock()

	for _, attr := range highCardinalityAttrs {
		d.opts.Logger.Warn(
			ctx,
			"instrument has high cardinality for attribute",
			log.String("instrument_name", d.name),
			log.String("attribute_name", attr),
			log.Int("max_cardinality", d.opts.MaxCardinality),
			log.Int("attributes_size", attrsLn),
		)
	}
}

// CheckAttrs implements HighCardinalityDetector.
func (d *detector) CheckAttrs(ctx context.Context, attrs []attribute.KeyValue) bool {
	if ctx == nil {
		ctx = context.Background()
	}

	d.mu.Lock()
	ok := true
	var details []log.Attr
	for _, attr := range attrs {
		if ok, details = d.check(string(attr.Key), attr.Value.Emit()); !ok {
			break
		}
	}
	d.mu.Unlock()

	if len(details) > 0 {
		d.opts.Logger.Warn(
			ctx,
			"instrument has high cardinality for attribute",
			details...,
		)
	}

	return ok
}

// Check implements HighCardinalityDetector.
func (d *detector) check(key string, value string) (bool, []log.Attr) {
	if uniqueValues, ok := d.attrs[key]; ok {
		if _, ok := uniqueValues[value]; ok {
			return true, nil
		}

		_, hasHighCardinality := d.highCardinality[key]
		if hasHighCardinality {
			return false, nil
		}

		ok := len(uniqueValues) < d.opts.MaxCardinality
		var details []log.Attr
		if ok {
			uniqueValues[value] = struct{}{}
		} else if !hasHighCardinality {
			d.highCardinality[key] = struct{}{}

			details = []log.Attr{
				log.String("instrument_name", d.name),
				log.String("attribute_name", key),
				log.Int("max_cardinality", d.opts.MaxCardinality),
				log.Int("attributes_size", len(d.attrs)),
				log.String("last_value", value),
			}
		}

		return ok, details
	}

	d.attrs[key] = map[string]struct{}{value: {}}

	return true, nil
}

// Shutdown implements CardinalityDetector.
func (d *detector) Shutdown() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return
	}
	d.closed = true

	if d.checkTicker != nil {
		d.checkTicker.Stop()
		close(d.checkDone)
	}
}

type noopDetector struct{}

// Shutdown implements CardinalityDetector.
func (*noopDetector) Shutdown() {}

// CheckAttrs implements HighCardinalityDetector.
func (*noopDetector) CheckAttrs(_ context.Context, _ []attribute.KeyValue) bool {
	return true
}
