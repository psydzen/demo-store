package analysisfixtures

import (
	"context"
	"log/slog"

	"github.com/spndxyz/quiz/internal/logtag"
	"github.com/spndxyz/quiz/internal/payments"
)

func sensitiveInLogTagBadPAN(ctx context.Context, card payments.Card) {
	// ruleid: go-sensitive-data-in-log-tags
	ctx = logtag.With(ctx, "pan", card.PAN)
	logtag.From(ctx).Info("charging card")
}

func sensitiveInLogTagBadCVV(ctx context.Context, card payments.Card) {
	// ruleid: go-sensitive-data-in-log-tags
	logtag.From(ctx).Info("charging card", "cvv", card.CVV)
}

func sensitiveInLogTagBadSlog(card payments.Card) {
	// ruleid: go-sensitive-data-in-log-tags
	slog.Info("charging card", "holder", card.Holder, "pan", card.PAN)
}

func sensitiveInLogTagBadPayer(ctx context.Context, payer payments.Payer) {
	// ruleid: go-sensitive-data-in-log-tags
	ctx = logtag.With(ctx, "email", payer.Email)
	// ruleid: go-sensitive-data-in-log-tags
	ctx = logtag.With(ctx, "tax_id", payer.TaxID)
	logtag.From(ctx).Info("payer resolved")
}

func sensitiveInLogTagOKMasked(ctx context.Context, card payments.Card) {
	// ok: go-sensitive-data-in-log-tags
	ctx = logtag.With(ctx, "card", card.Masked())
	logtag.From(ctx).Info("charging card")
}

func sensitiveInLogTagOKLast4(ctx context.Context, p payments.Payment) {
	// ok: go-sensitive-data-in-log-tags
	ctx = logtag.With(ctx, "card_last4", p.CardLast4)
	logtag.From(ctx).Info("payment stored")
}
