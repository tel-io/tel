package global

import (
	"github.com/tel-io/tel/v2/pkg/log"
)

func Handle(err error, msg string) {
	if err != nil {
		Error(err, msg, log.AttrCallerSkipOffset(1))
	}
}
