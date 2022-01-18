package zlogfmt

import (
	"regexp"
)

var lokiRegExp = regexp.MustCompile(`[.\s]`)

func lokiKeyMutator(key *string) {
	*key = lokiRegExp.ReplaceAllString(*key, "_")
}
