package observability

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

type traceContextHandler struct {
	slog.Handler
}

// Logger returns a structured JSON slog.Logger that injects trace_id and span_id
// when a trace context is present.
func Logger() *slog.Logger {
	return slog.New(&traceContextHandler{
		Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	})
}

func (h *traceContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", spanContext.TraceID().String()),
			slog.String("span_id", spanContext.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func (h *traceContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *traceContextHandler) WithGroup(name string) slog.Handler {
	return &traceContextHandler{Handler: h.Handler.WithGroup(name)}
}
