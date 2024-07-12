package metric

import (
	"context"

	"github.com/tel-io/tel/v2/pkg/cardinalitydetector"
	"go.opentelemetry.io/otel/metric"
)

var _ metric.Observer = (*cdObserver)(nil)

type cdObserver struct {
	metric.Observer
	stopCtx context.Context
}

func (o *cdObserver) ObserveFloat64(obsrv metric.Float64Observable, value float64, options ...metric.ObserveOption) {
	attrs := metric.NewObserveConfig(options).Attributes()

	var cardinalityDetector cardinalitydetector.Detector
	switch t := obsrv.(type) {
	case *cdFloat64ObservableCounter:
		cardinalityDetector = t.cardinalityDetector
	case *cdFloat64ObservableUpDownCounter:
		cardinalityDetector = t.cardinalityDetector
	case *cdFloat64ObservableGauge:
		cardinalityDetector = t.cardinalityDetector
	}

	unwrapped := obsrv
	if wrapped, ok := obsrv.(unwrapper); ok {
		unwrapped = wrapped.Unwrap().(metric.Float64Observable) //nolint:forcetypeassert
	}

	if cardinalityDetector == nil {
		o.Observer.ObserveFloat64(unwrapped, value, options...)

		return
	}

	if cardinalityDetector.CheckAttrs(o.stopCtx, attrs.ToSlice()) {
		o.Observer.ObserveFloat64(unwrapped, value, options...)
	}
}

func (o *cdObserver) ObserveInt64(obsrv metric.Int64Observable, value int64, options ...metric.ObserveOption) {
	attrs := metric.NewObserveConfig(options).Attributes()

	var cardinalityDetector cardinalitydetector.Detector
	switch t := obsrv.(type) {
	case *cdInt64ObservableCounter:
		cardinalityDetector = t.cardinalityDetector
	case *cdInt64ObservableUpDownCounter:
		cardinalityDetector = t.cardinalityDetector
	case *cdInt64ObservableGauge:
		cardinalityDetector = t.cardinalityDetector
	}

	unwrapped := obsrv
	if wrapped, ok := obsrv.(unwrapper); ok {
		unwrapped = wrapped.Unwrap().(metric.Int64Observable) //nolint:forcetypeassert
	}

	if cardinalityDetector == nil {
		o.Observer.ObserveInt64(unwrapped, value, options...)

		return
	}

	if cardinalityDetector.CheckAttrs(o.stopCtx, attrs.ToSlice()) {
		o.Observer.ObserveInt64(unwrapped, value, options...)
	}
}

type cdFloat64Observer struct {
	metric.Float64Observer
	cardinalityDetector cardinalitydetector.Detector
	stopCtx             context.Context
}

func (o *cdFloat64Observer) Observe(value float64, options ...metric.ObserveOption) {
	attrs := metric.NewObserveConfig(options).Attributes()
	if o.cardinalityDetector.CheckAttrs(o.stopCtx, attrs.ToSlice()) {
		o.Float64Observer.Observe(value, options...)
	}
}

type cdInt64Observer struct {
	metric.Int64Observer
	cardinalityDetector cardinalitydetector.Detector
	stopCtx             context.Context
}

func (o *cdInt64Observer) Observe(value int64, options ...metric.ObserveOption) {
	attrs := metric.NewObserveConfig(options).Attributes()
	if o.cardinalityDetector.CheckAttrs(o.stopCtx, attrs.ToSlice()) {
		o.Int64Observer.Observe(value, options...)
	}
}
