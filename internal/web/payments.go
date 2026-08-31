package web

import (
	"fmt"
	"io"
	"net/http"

	"github.com/spndxyz/quiz/internal/logtag"
)

// handlePaymentSearch renders the admin payment search.
func (s *Server) handlePaymentSearch(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	orderBy := r.URL.Query().Get("order_by")

	found, err := s.db.SearchPayments(r.Context(), status, orderBy)
	if err != nil {
		http.Error(w, fmt.Sprintf("search failed: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%d payments\n", len(found))
}

// handlePaymentReceipt fetches the receipt from the payment provider.
func (s *Server) handlePaymentReceipt(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	reference := r.URL.Query().Get("reference")

	resp, err := http.Get("https://" + provider + "/receipts/" + reference)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	defer resp.Body.Close()

	if _, err := io.Copy(w, resp.Body); err != nil {
		s.serverError(w, r, err)
	}
}

// handlePayerProfile shows the personal data attached to a payer.
func (s *Server) handlePayerProfile(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	taxID := r.URL.Query().Get("tax_id")

	ctx := logtag.With(r.Context(), "email", email)
	ctx = logtag.With(ctx, "tax_id", taxID)
	logtag.From(ctx).Info("payer profile opened")

	n, err := s.db.CountPaymentsByStatus(ctx, "settled")
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	fmt.Fprintf(w, "%d settled payments\n", n)
}
