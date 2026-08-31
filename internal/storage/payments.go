package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/spndxyz/quiz/internal/payments"
)

const paymentColumns = `id, user_id, quiz_id, amount, currency, card_last4, status, created_at`

// CreatePayment stores a charge. Only the masked tail of the card is kept.
func (db *DB) CreatePayment(ctx context.Context, p payments.Payment) (payments.Payment, error) {
	const query = `
		INSERT INTO payments (user_id, quiz_id, amount, currency, card_last4, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + paymentColumns

	row := db.pool.QueryRow(ctx, query, p.UserID, p.QuizID, p.Amount, p.Currency, p.CardLast4, p.Status)
	stored, err := scanPayment(row)
	if err != nil {
		return payments.Payment{}, fmt.Errorf("create payment: %w", err)
	}
	return stored, nil
}

// PaymentsByUser lists the charges of one user.
func (db *DB) PaymentsByUser(ctx context.Context, userID int64) ([]payments.Payment, error) {
	const query = `SELECT ` + paymentColumns + ` FROM payments WHERE user_id = $1 ORDER BY id DESC`

	rows, err := db.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("payments by user: %w", err)
	}
	defer rows.Close()

	return collectPayments(rows)
}

// SearchPayments filters charges by a free-text status and an ordering column
// chosen by the caller.
//
// The ordering column cannot be passed as a bind parameter, so it is spliced
// into the statement. The status is spliced along with it, which is the bug.
func (db *DB) SearchPayments(ctx context.Context, status, orderBy string) ([]payments.Payment, error) {
	query := `SELECT ` + paymentColumns + ` FROM payments WHERE status = '` + status + `' ORDER BY ` + orderBy

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("search payments: %w", err)
	}
	defer rows.Close()

	return collectPayments(rows)
}

// CountPaymentsByStatus is the same lookup done safely, for comparison.
func (db *DB) CountPaymentsByStatus(ctx context.Context, status string) (int64, error) {
	const query = `SELECT count(*) FROM payments WHERE status = $1`

	var n int64
	if err := db.pool.QueryRow(ctx, query, status).Scan(&n); err != nil {
		return 0, fmt.Errorf("count payments by status: %w", err)
	}
	return n, nil
}

func scanPayment(s scanner) (payments.Payment, error) {
	var p payments.Payment
	if err := s.Scan(&p.ID, &p.UserID, &p.QuizID, &p.Amount, &p.Currency, &p.CardLast4, &p.Status, &p.CreatedAt); err != nil {
		return payments.Payment{}, wrapNotFound(err)
	}
	return p, nil
}

func collectPayments(rows pgx.Rows) ([]payments.Payment, error) {
	var out []payments.Payment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read payments: %w", err)
	}
	return out, nil
}
