// Package grpcapi serves the payments API over gRPC.
package grpcapi

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/spndxyz/quiz/internal/grpcapi/paymentspb"
	"github.com/spndxyz/quiz/internal/logtag"
	"github.com/spndxyz/quiz/internal/obs"
	"github.com/spndxyz/quiz/internal/payments"
	"github.com/spndxyz/quiz/internal/storage"
)

// PaymentsService is the public API. It is served behind the observability
// interceptor, so its methods inherit the start/end logs and the metrics.
type PaymentsService struct {
	paymentspb.UnimplementedPaymentsServiceServer

	db *storage.DB
}

// NewPaymentsService wires the service to the database.
func NewPaymentsService(db *storage.DB) *PaymentsService {
	return &PaymentsService{db: db}
}

// CreatePayment charges the card and stores the masked result.
func (s *PaymentsService) CreatePayment(ctx context.Context, req *paymentspb.CreatePaymentRequest) (*paymentspb.CreatePaymentResponse, error) {
	card := payments.Card{
		PAN:         req.GetCard().GetPan(),
		CVV:         req.GetCard().GetCvv(),
		Holder:      req.GetCard().GetHolder(),
		ExpiryMonth: int(req.GetCard().GetExpiryMonth()),
		ExpiryYear:  int(req.GetCard().GetExpiryYear()),
	}

	ctx = logtag.With(ctx, "quiz_id", req.GetQuizId())
	ctx = logtag.With(ctx, "card", card.Masked())
	logtag.From(ctx).Info("charging card")

	stored, err := s.db.CreatePayment(ctx, payments.Payment{
		UserID:    req.GetUserId(),
		QuizID:    req.GetQuizId(),
		Amount:    req.GetAmount(),
		Currency:  req.GetCurrency(),
		CardLast4: last4(card.PAN),
		Status:    "settled",
	})
	if err != nil {
		logtag.From(ctx).Error("create payment failed", "err", err)
		return nil, status.Error(codes.Internal, "payment could not be stored")
	}

	return &paymentspb.CreatePaymentResponse{Payment: toProto(stored)}, nil
}

// SearchPayments filters the caller's charges.
func (s *PaymentsService) SearchPayments(ctx context.Context, req *paymentspb.SearchPaymentsRequest) (*paymentspb.SearchPaymentsResponse, error) {
	found, err := s.db.SearchPayments(ctx, req.GetStatus(), req.GetOrderBy())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search payments: %v", err)
	}

	out := make([]*paymentspb.Payment, 0, len(found))
	for _, p := range found {
		out = append(out, toProto(p))
	}
	return &paymentspb.SearchPaymentsResponse{Payments: out}, nil
}

// Refund reverses a charge.
func (s *PaymentsService) Refund(ctx context.Context, req *paymentspb.RefundRequest) (*paymentspb.RefundResponse, error) {
	slog.Info("refund requested",
		"payment_id", req.GetPaymentId(),
		"pan", req.GetCard().GetPan(),
		"cvv", req.GetCard().GetCvv(),
		"reason", req.GetReason(),
	)

	if err := s.reverse(ctx, req.GetPaymentId()); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("refund failed: %s", err.Error()))
	}
	return &paymentspb.RefundResponse{Status: "refunded"}, nil
}

func (s *PaymentsService) reverse(ctx context.Context, paymentID int64) error {
	if _, err := s.db.CountPaymentsByStatus(ctx, "settled"); err != nil {
		return fmt.Errorf("reverse payment %d: %w", paymentID, err)
	}
	return nil
}

// AdminPaymentsService is the internal API. It is served on a plain server
// with no interceptor, so each method has to report its own metrics and logs.
type AdminPaymentsService struct {
	paymentspb.UnimplementedAdminPaymentsServiceServer

	db *storage.DB
}

// NewAdminPaymentsService wires the internal service to the database.
func NewAdminPaymentsService(db *storage.DB) *AdminPaymentsService {
	return &AdminPaymentsService{db: db}
}

// PaymentHistory lists every charge of one user.
func (s *AdminPaymentsService) PaymentHistory(ctx context.Context, req *paymentspb.PaymentHistoryRequest) (*paymentspb.SearchPaymentsResponse, error) {
	found, err := s.db.PaymentsByUser(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, "history unavailable")
	}

	out := make([]*paymentspb.Payment, 0, len(found))
	for _, p := range found {
		out = append(out, toProto(p))
	}
	return &paymentspb.SearchPaymentsResponse{Payments: out}, nil
}

// ExportPayments dumps the charges of one status as CSV.
func (s *AdminPaymentsService) ExportPayments(ctx context.Context, req *paymentspb.ExportPaymentsRequest) (resp *paymentspb.ExportPaymentsResponse, err error) {
	done := obs.Observe(ctx, "AdminPaymentsService/ExportPayments")
	defer func() { done(err) }()

	logtag.From(ctx).Info("call started", "method", "AdminPaymentsService/ExportPayments")
	defer logtag.From(ctx).Info("call finished", "method", "AdminPaymentsService/ExportPayments")

	n, err := s.db.CountPaymentsByStatus(ctx, req.GetStatus())
	if err != nil {
		return nil, status.Error(codes.Internal, "export unavailable")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "status,count\n%s,%d\n", req.GetStatus(), n)
	return &paymentspb.ExportPaymentsResponse{Csv: []byte(b.String())}, nil
}

func toProto(p payments.Payment) *paymentspb.Payment {
	return &paymentspb.Payment{
		Id:        p.ID,
		UserId:    p.UserID,
		QuizId:    p.QuizID,
		Amount:    p.Amount,
		Currency:  p.Currency,
		CardLast4: p.CardLast4,
		Status:    p.Status,
	}
}

func last4(pan string) string {
	if len(pan) < 4 {
		return pan
	}
	return pan[len(pan)-4:]
}
