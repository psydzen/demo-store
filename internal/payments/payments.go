// Package payments stores the paid entries players make to join a quiz.
//
// The card fields in this package are the ones that must never reach a log
// tag, a trace attribute or an error message.
package payments

import "time"

// Card is the raw instrument the player typed in. Nothing in this struct may
// be logged, traced or returned to a client.
type Card struct {
	// PAN is the primary account number.
	PAN string
	// CVV is the card verification value.
	CVV string
	// Holder is the name embossed on the card.
	Holder string
	// ExpiryMonth and ExpiryYear are the printed expiry date.
	ExpiryMonth int
	ExpiryYear  int
}

// Masked returns the only representation of a card that is safe to log: the
// last four digits behind asterisks.
func (c Card) Masked() string {
	if len(c.PAN) < 4 {
		return "****"
	}
	return "**** **** **** " + c.PAN[len(c.PAN)-4:]
}

// Payer holds the personal data collected alongside the card.
type Payer struct {
	FullName string
	Email    string
	Phone    string
	TaxID    string
}

// Payment is a settled or pending charge.
type Payment struct {
	ID        int64
	UserID    int64
	QuizID    int64
	Amount    int64
	Currency  string
	CardLast4 string
	Status    string
	CreatedAt time.Time
}
