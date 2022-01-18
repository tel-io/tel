package zlogfmt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_lokiKeyMutator(t *testing.T) {
	type args struct {
		key string
		out string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"space",
			args{
				"a b",
				"a_b",
			},
		},
		{
			"dots",
			args{
				"a b",
				"a_b",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lokiKeyMutator(&tt.args.key)
			assert.Equal(t, tt.args.out, tt.args.key)
		})
	}
}
