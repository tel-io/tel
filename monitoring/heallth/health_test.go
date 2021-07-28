package health

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealth(t *testing.T) {
	h := NewHealth()
	h.Set(UP)
	h.AddInfo("version", "latest")

	func(h Health) {
		data, err := json.Marshal(h)
		assert.NoError(t, err)

		assert.Contains(t, string(data), "version")
		assert.Contains(t, string(data), "status")
	}(h)

}
