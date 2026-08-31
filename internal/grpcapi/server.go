package grpcapi

import (
	"log/slog"

	"google.golang.org/grpc"

	"github.com/spndxyz/quiz/internal/grpcapi/paymentspb"
	"github.com/spndxyz/quiz/internal/storage"
)

// NewPublicServer builds the player-facing gRPC server. Everything registered
// here runs behind the observability interceptor.
func NewPublicServer(db *storage.DB, log *slog.Logger) *grpc.Server {
	srv := grpc.NewServer(grpc.UnaryInterceptor(Observability(log)))
	paymentspb.RegisterPaymentsServiceServer(srv, NewPaymentsService(db))
	return srv
}

// NewAdminServer builds the internal gRPC server. It is reachable only from
// the private network, so it is registered without the interceptor.
func NewAdminServer(db *storage.DB) *grpc.Server {
	srv := grpc.NewServer()
	paymentspb.RegisterAdminPaymentsServiceServer(srv, NewAdminPaymentsService(db))
	return srv
}
