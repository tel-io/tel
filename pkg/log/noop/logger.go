//nolint:revive
package noop

import (
	"context"

	"github.com/tel-io/tel/v2/pkg/log"
	"go.opentelemetry.io/otel/trace"
)

func NewLogger() *Logger {
	return &Logger{}
}

type Logger struct{}

func (l *Logger) LogAttrs(ctx context.Context, level log.Level, msg string, attrs ...log.Attr) {}
func (l *Logger) Enabled(level log.Level) bool                                                 { return false }
func (l *Logger) Debug(ctx context.Context, msg string, attrs ...log.Attr)                     {}
func (l *Logger) Info(ctx context.Context, msg string, attrs ...log.Attr)                      {}
func (l *Logger) Warn(ctx context.Context, msg string, attrs ...log.Attr)                      {}
func (l *Logger) Error(ctx context.Context, msg string, attrs ...log.Attr)                     {}
func (l *Logger) Panic(ctx context.Context, msg string, attrs ...log.Attr)                     {}
func (l *Logger) Fatal(ctx context.Context, msg string, attrs ...log.Attr)                     {}
func (l *Logger) With(attrs ...log.Attr) log.Logger                                            { return l }
func (l *Logger) Named(name string) log.Logger                                                 { return l }
func (l *Logger) NewContext(ctx context.Context) context.Context                               { return ctx }
func (l *Logger) For(ctx context.Context) log.Logger                                           { return l }
func (l *Logger) ForSpan(span trace.Span) log.Logger                                           { return l }
func (l *Logger) Attrs() []log.Attr                                                            { return nil }
func (l *Logger) Span() trace.Span                                                             { return nil }
func (l *Logger) Handler() log.Handler                                                         { return nil }
