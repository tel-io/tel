package otelerr

import (
	"reflect"
	"testing"

	"go.uber.org/zap"
)

func Test_conv(t *testing.T) {
	type args struct {
		v []interface{}
	}
	tests := []struct {
		name string
		args args
		want []zap.Field
	}{
		{
			"OK",
			args{[]interface{}{
				"name", "github.com/tel-io/instrumentation/plugins/otelsql",
				"version", "0.3.4",
				"url", "https://opentelemetry.io/schemas/1.10.0",
			}},
			[]zap.Field{
				zap.String("name", "github.com/tel-io/instrumentation/plugins/otelsql"),
				zap.String("version", "0.3.4"),
				zap.String("url", "https://opentelemetry.io/schemas/1.10.0"),
			},
		},
		{
			"not equal",
			args{[]interface{}{
				"name", "github.com/tel-io/instrumentation/plugins/otelsql",
				"version", "0.3.4",
				"url",
			}},
			[]zap.Field{
				zap.String("name", "github.com/tel-io/instrumentation/plugins/otelsql"),
				zap.String("version", "0.3.4"),
			},
		},
		{
			"not string value",
			args{[]interface{}{
				"name", "github.com/tel-io/instrumentation/plugins/otelsql",
				"version", 0.3,
				"url", "https://opentelemetry.io/schemas/1.10.0",
			}},
			[]zap.Field{
				zap.String("name", "github.com/tel-io/instrumentation/plugins/otelsql"),
				zap.String("url", "https://opentelemetry.io/schemas/1.10.0"),
			},
		},
		{
			"not string key and value",
			args{[]interface{}{
				"name", "github.com/tel-io/instrumentation/plugins/otelsql",
				true, 0.2,
				"url", "https://opentelemetry.io/schemas/1.10.0",
			}},
			[]zap.Field{
				zap.String("name", "github.com/tel-io/instrumentation/plugins/otelsql"),
				zap.String("url", "https://opentelemetry.io/schemas/1.10.0"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := conv(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("conv() = %v, want %v", got, tt.want)
			}
		})
	}
}
