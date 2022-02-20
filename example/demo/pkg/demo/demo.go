package demo

import (
	"database/sql"
	"runtime/debug"

	"github.com/pkg/errors"
)

func StackTrace() string {
	return string(debug.Stack())
}

func E() error {
	return errors.WithStack(MakeError())
}

func MakeError() error {
	return errors.WithStack(sql.ErrNoRows)
}
