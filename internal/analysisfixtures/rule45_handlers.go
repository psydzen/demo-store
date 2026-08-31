package analysisfixtures

import (
	"context"
	"net/http"

	"google.golang.org/grpc"

	"github.com/spndxyz/quiz/internal/grpcapi/paymentspb"
	"github.com/spndxyz/quiz/internal/logtag"
	"github.com/spndxyz/quiz/internal/obs"
)

// fixtureService stands in for a generated gRPC service implementation. It is
// registered on a server built without an interceptor, so each of its methods
// has to report its own metrics and its own start/end logs.
type fixtureService struct {
	paymentspb.UnimplementedAdminPaymentsServiceServer
}

func newFixtureServer() *grpc.Server {
	srv := grpc.NewServer()
	paymentspb.RegisterAdminPaymentsServiceServer(srv, &fixtureService{})
	return srv
}

// ruleid: go-handler-without-metrics
// ruleid: go-handler-without-start-end-log
func (s *fixtureService) PaymentHistory(ctx context.Context, req *paymentspb.PaymentHistoryRequest) (*paymentspb.SearchPaymentsResponse, error) {
	_ = req.GetUserId()
	return &paymentspb.SearchPaymentsResponse{}, nil
}

// ok: go-handler-without-metrics
// ok: go-handler-without-start-end-log
func (s *fixtureService) ExportPayments(ctx context.Context, req *paymentspb.ExportPaymentsRequest) (resp *paymentspb.ExportPaymentsResponse, err error) {
	done := obs.Observe(ctx, "AdminPaymentsService/ExportPayments")
	defer func() { done(err) }()

	logtag.From(ctx).Info("call started", "method", "AdminPaymentsService/ExportPayments")
	defer logtag.From(ctx).Info("call finished", "method", "AdminPaymentsService/ExportPayments")

	_ = req.GetStatus()
	return &paymentspb.ExportPaymentsResponse{}, nil
}

// ruleid: go-handler-without-metrics
// ruleid: go-handler-without-start-end-log
func fixtureHTTPHandlerBad(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("ok"))
}

// ok: go-handler-without-metrics
// ok: go-handler-without-start-end-log
func fixtureHTTPHandlerOK(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	done := obs.Observe(ctx, "GET /fixture")
	logtag.From(ctx).Info("call started", "method", "GET /fixture")

	_, err := w.Write([]byte("ok"))

	logtag.From(ctx).Info("call finished", "method", "GET /fixture")
	done(err)
}

// ruleid: go-handler-without-start-end-log
func fixtureHTTPHandlerStartOnly(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	done := obs.Observe(ctx, "GET /fixture/start-only")
	logtag.From(ctx).Info("call started", "method", "GET /fixture/start-only")

	_, err := w.Write([]byte("ok"))
	done(err)
}
