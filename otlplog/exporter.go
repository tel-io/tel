package otlplog

import (
	"context"
	"errors"
	"sync"

	"github.com/tel-io/tel/v2/otlplog/logskd"
	"github.com/tel-io/tel/v2/pkg/logtransform"
	"go.opentelemetry.io/otel/sdk/resource"
)

var (
	errAlreadyStarted = errors.New("already started")
)

// Exporter exports trace data in the OTLP wire format.
type Exporter struct {
	client Client
	res    *resource.Resource

	mu      sync.RWMutex
	started bool

	startOnce sync.Once
	stopOnce  sync.Once
}

// ExportLogs exports a batch of spans.
func (e *Exporter) ExportLogs(ctx context.Context, in []logskd.Log) error {
	if len(in) == 0 {
		return nil
	}

	return e.client.UploadLogs(ctx, logtransform.Trans(e.res, in))
}

var _ logskd.Exporter = &Exporter{}

// Start establishes a connection to the receiving endpoint.
func (e *Exporter) Start(ctx context.Context) error {
	var err = errAlreadyStarted
	e.startOnce.Do(func() {
		e.mu.Lock()
		e.started = true
		e.mu.Unlock()
		err = e.client.Start(ctx)
	})

	return err
}

// Shutdown flushes all exports and closes all connections to the receiving endpoint.
func (e *Exporter) Shutdown(ctx context.Context) error {
	e.mu.RLock()
	started := e.started
	e.mu.RUnlock()

	if !started {
		return nil
	}

	var err error

	e.stopOnce.Do(func() {
		err = e.client.Stop(ctx)
		e.mu.Lock()
		e.started = false
		e.mu.Unlock()
	})

	return err
}

// New constructs a new Exporter and starts it.
func New(ctx context.Context, client Client, res *resource.Resource) (*Exporter, error) {
	exp := NewUnstarted(client, res)
	if err := exp.Start(ctx); err != nil {
		return nil, err
	}
	return exp, nil
}

// NewUnstarted constructs a new Exporter and does not start it.
func NewUnstarted(client Client, res *resource.Resource) *Exporter {
	return &Exporter{
		client: client,
		res:    res,
	}
}
