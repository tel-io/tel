package pgx

import (
	"bufio"
	"fmt"
	"testing"

	"github.com/d7561985/tel/v2"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		name  string
		level pgx.LogLevel
		data  map[string]interface{}
		check []string
	}{
		{
			"LogLevelTrace",
			pgx.LogLevelTrace,
			map[string]interface{}{"X": true},
			[]string{"PGX_LOG_LEVEL", "trace"},
		},
		{
			"LogLevelDebug",
			pgx.LogLevelDebug,
			map[string]interface{}{"PGX_LOG_LEVEL": true},
			[]string{"PGX_LOG_LEVEL", "true"},
		},
		{
			"LogLevelInfo",
			pgx.LogLevelInfo,
			map[string]interface{}{"PGX_LOG_LEVEL": true},
			[]string{"PGX_LOG_LEVEL", "true"},
		},
		{
			"LogLevelWarn",
			pgx.LogLevelWarn,
			map[string]interface{}{"PGX_LOG_LEVEL": true},
			[]string{"PGX_LOG_LEVEL", "true"},
		},
		{
			"LogLevelError",
			pgx.LogLevelError,
			map[string]interface{}{"PGX_LOG_LEVEL": true},
			[]string{"PGX_LOG_LEVEL", "true"},
		},
		{
			"check sql and args fields",
			pgx.LogLevelInfo,
			map[string]interface{}{fSql: "insert * from table where user = $1", fArgs: []interface{}{100500}},
			[]string{"insert * from table where user = 100500"},
		},
		{
			"check sql no args",
			pgx.LogLevelInfo,
			map[string]interface{}{fSql: "insert * from table where user = $1"},
			[]string{"insert * from table where user = $"},
		},
		{
			"multi-line",
			pgx.LogLevelInfo,
			map[string]interface{}{fSql: `UPDATE tx SET revert = true WHERE
						created_at < current_timestamp  AND  created_at > current_timestamp - interval '3' month AND
						id = $1 AND "accountId" = $2 `},
			[]string{"UPDATE tx SET revert = true WHERE created_at < current_timestamp AND created_at > current_timestamp - interval"},
		},
	}

	tt := tel.NewNull()

	buf := tel.SetLogOutput(&tt)
	ctx := tt.Ctx()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			NewLogger().Log(ctx, test.level, "test", test.data)

			line, _, err := bufio.NewReader(buf).ReadLine()
			assert.NoError(t, err)
			fmt.Println(string(line))

			for _, val := range test.check {
				assert.Contains(t, string(line), val)
			}
		})
	}
}
