// Package logging provides structured JSON logging and request tracing for
// the TrustCheck API. Every request receives a request ID, exposed both in
// the request context and as the X-Request-ID response header, and emits
// exactly one JSON log line when it completes.
package logging

import (
	"log/slog"
	"os"
	"time"
)

// New returns a structured JSON logger writing to standard output.
// Timestamps are rendered as RFC 3339 with the "timestamp" key to match the
// documented log format.
func New() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Key = "timestamp"
				a.Value = slog.StringValue(a.Value.Time().UTC().Format(time.RFC3339))
			}
			return a
		},
	})
	return slog.New(handler)
}
