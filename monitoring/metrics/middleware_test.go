package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegRep(t *testing.T) {
	var tests = []struct {
		in  string
		out string
	}{
		{"/userdetails/44831", "/userdetails/:ID"},
		{"/v1/userdetails/44831", "/v1/userdetails/:ID"},
		{"/v1/userdetails/44831/send", "/v1/userdetails/:ID/send"},
	}

	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			assert.Equal(t, test.out, urlPrepare(test.in))
		})
	}
}
