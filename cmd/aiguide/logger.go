package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
)

func initLogger() {
	opts := &slog.HandlerOptions{
		AddSource: true,
	}
	// Use a custom writer to convert escaped "\n" back to real newlines
	writer := &newlineWriter{w: os.Stdout}
	var handler slog.Handler = slog.NewTextHandler(writer, opts)
	handler = &stackHandler{handler}
	slog.SetDefault(slog.New(handler))
}

type newlineWriter struct {
	w io.Writer
}

func (w *newlineWriter) Write(p []byte) (int, error) {
	// slog.TextHandler escapes newlines and tabs as literal strings "\\n" and "\\t"
	// We convert them back to real characters for readability
	s := strings.ReplaceAll(string(p), `\n`, "\n")
	s = strings.ReplaceAll(s, `\t`, "\t")
	_, err := w.w.Write([]byte(s))
	return len(p), err
}

type stackHandler struct {
	slog.Handler
}

func (h *stackHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		buf := make([]byte, 2048)
		n := runtime.Stack(buf, false)
		r.AddAttrs(slog.String("stack", string(buf[:n])))
	}
	return h.Handler.Handle(ctx, r)
}
