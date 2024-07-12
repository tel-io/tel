package log

import (
	"os"

	"log/slog"
)

type Handler = slog.Handler
type TextHandler = slog.TextHandler
type JSONHandler = slog.JSONHandler

func NewTextHandler(minLevel slog.Leveler, replacerAttrs ...ReplacerAttr) *TextHandler {
	replacerAttr := ChainReplacerAttrs(append([]ReplacerAttr{defaultReplaceAttr}, replacerAttrs...))

	return slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			AddSource:   AddSource,
			Level:       minLevel,
			ReplaceAttr: replacerAttr,
		},
	)
}

func NewJSONHandler(minLevel slog.Leveler, replacerAttrs ...ReplacerAttr) *JSONHandler {
	replacerAttr := ChainReplacerAttrs(append([]ReplacerAttr{defaultReplaceAttr}, replacerAttrs...))

	return slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			AddSource:   AddSource,
			Level:       minLevel,
			ReplaceAttr: replacerAttr,
		},
	)
}
