package analysisfixtures

import (
	"context"
	"log/slog"

	"github.com/spndxyz/quiz/internal/logtag"
)

func loggerWithoutContextBad(ctx context.Context, quizID int64) {
	_ = ctx
	// ruleid: go-logger-without-logtag-context
	slog.Info("quiz opened", "quiz_id", quizID)
}

func loggerWithoutContextBadDefault(ctx context.Context, quizID int64) {
	_ = ctx
	// ruleid: go-logger-without-logtag-context
	slog.Default().Error("quiz failed", "quiz_id", quizID)
}

func loggerWithoutContextBadInstance(ctx context.Context, log *slog.Logger, quizID int64) {
	_ = ctx
	// ruleid: go-logger-without-logtag-context
	log.Info("quiz opened", "quiz_id", quizID)
}

func loggerWithContextOK(ctx context.Context, quizID int64) {
	ctx = logtag.With(ctx, "quiz_id", quizID)
	// ok: go-logger-without-logtag-context
	logtag.From(ctx).Info("quiz opened")
}

// loggerNoContextAvailableOK logs at startup, where no request context exists.
func loggerNoContextAvailableOK(log *slog.Logger) {
	// ok: go-logger-without-logtag-context
	log.Info("migrations applied")
}
