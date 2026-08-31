package grpcapi

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"

	"github.com/spndxyz/quiz/internal/logtag"
	"github.com/spndxyz/quiz/internal/obs"
)

// Observability returns the interceptor every public RPC runs behind: it puts
// a tagged logger into the context, logs the start and the end of the call and
// reports the request metrics.
func Observability(base *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx = logtag.Into(ctx, logtag.New(base))
		ctx = logtag.With(ctx, "method", info.FullMethod)

		done := obs.Observe(ctx, info.FullMethod)

		logtag.From(ctx).Info("call started")
		resp, err := handler(ctx, req)
		logtag.From(ctx).Info("call finished", "err", err)

		done(err)
		return resp, err
	}
}
