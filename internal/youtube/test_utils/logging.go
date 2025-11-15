package test_utils

import (
	"io"
	"log/slog"
)

func NewNopLogger() *slog.Logger {
	handler := slog.NewTextHandler(io.Discard, nil)
	return slog.New(handler)
}
