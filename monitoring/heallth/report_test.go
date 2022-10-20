package health

import (
	"context"
	"encoding/json"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealth(t *testing.T) {
	h := NewReport("demo", true)
	h.AddInfo(attribute.Key("version").String("latest"))

	func(h *Report) {
		data, err := json.Marshal(h)
		assert.NoError(t, err)
		fmt.Println(string(data))

		assert.Contains(t, string(data), "version")
		assert.Contains(t, string(data), "online")
	}(h)

	t.Run("xxx", func(t *testing.T) {
		c := NewSimple(CheckerFunc(func(ctx context.Context) ReportDocument {
			return h
		}))

		b := c.Check(context.Background())
		x, err := json.Marshal(b)
		fmt.Println(string(x), err)
	})

}
