package tel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// check whole context stack: WithContext, updateContext, FromCtx
func Test_telemetry_With(t *testing.T) {
	ctx := NewNull().Ctx()
	buf := SetLogOutput(ctx)

	t.Run("note tele context", func(t *testing.T) {
		assert.NotPanics(t, func() {
			FromCtx(context.TODO()).Info("INFO MSG")
		})
	})

	t.Run("no injection", func(t *testing.T) {
		// check without injection
		FromCtx(ctx).Info("INFO MSG")
		assert.Contains(t, buf.String(), "INFO MSG")
		buf.Reset() // clean
	})

	t.Run("injection", func(t *testing.T) {
		// create tele-copy
		ctxInstance := FromCtx(ctx).Ctx()
		FromCtx(ctxInstance).PutFields(String("INJECTED STRING", "OK"))
		// print copy with injected fields
		FromCtx(ctxInstance).Info("INSTANCE ")

		assert.Contains(t, buf.String(), "INJECTED STRING")
		buf.Reset() // clean
	})

	// check if we not affect original ctx
	t.Run("check original context", func(t *testing.T) {
		FromCtx(ctx).Info("INFO MSG")
		assert.NotContains(t, buf.String(), "INJECTED STRING")
		buf.Reset() // clean
	})

	// StartSpanFromContext goal to check if return sctx save tele reference and save to correct stream
	t.Run("check span from context", func(t *testing.T) {
		span, sctx := StartSpanFromContext(ctx, "test")
		defer span.End()

		const testMsg = "traced log"

		FromCtx(sctx).Info(testMsg)
		assert.Contains(t, buf.String(), testMsg)
		buf.Reset() // clean
	})

	// StartSpan goal to check if return sctx save tele reference and save to correct stream
	t.Run("check new span", func(t *testing.T) {
		span, sctx := FromCtx(ctx).StartSpan("test")
		defer span.End()

		const testMsg = "traced log"

		FromCtx(sctx).Info(testMsg)
		assert.Contains(t, buf.String(), testMsg)
		buf.Reset() // clean
	})
}
