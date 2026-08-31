// Package logtag carries a request-scoped logger through context.Context.
//
// Every log line produced through this package inherits the tags attached
// earlier in the request, so a handler never has to thread the request id or
// the caller identity down by hand.
package logtag

import (
	"context"
	"log/slog"
)

// ctxKey keeps the logger out of the string namespace.
type ctxKey struct{}

// Logger wraps slog.Logger with the tags collected so far.
type Logger struct {
	base *slog.Logger
	tags []any
}

// New builds a Logger on top of an slog handler.
func New(base *slog.Logger) *Logger { return &Logger{base: base} }

// Into stores the logger in the context so that From can find it later.
func Into(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// From returns the logger stored in the context. It never returns nil: a
// request that lost its logger still logs, just without tags.
func From(ctx context.Context) *Logger {
	if l, ok := ctx.Value(ctxKey{}).(*Logger); ok && l != nil {
		return l
	}
	return &Logger{base: slog.Default()}
}

// With attaches a tag to the logger stored in the context and returns the
// derived context. Tags accumulate for the rest of the request.
func With(ctx context.Context, key string, value any) context.Context {
	l := From(ctx)
	next := &Logger{base: l.base, tags: append(append([]any{}, l.tags...), key, value)}
	return Into(ctx, next)
}

// Info logs at info level with every tag collected so far.
func (l *Logger) Info(msg string, args ...any) {
	l.base.Info(msg, append(append([]any{}, l.tags...), args...)...)
}

// Error logs at error level with every tag collected so far.
func (l *Logger) Error(msg string, args ...any) {
	l.base.Error(msg, append(append([]any{}, l.tags...), args...)...)
}
