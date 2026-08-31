package analysisfixtures

import (
	"net/http"

	"github.com/spndxyz/quiz/internal/payments"
)

// The helpers below are the sources. Their sinks live in crossfile_sink.go, so
// a single-file engine cannot connect the two.

// crossStatus reads a filter straight off the request.
func crossStatus(r *http.Request) string {
	return r.URL.Query().Get("status")
}

// crossProvider reads a host name straight off the request.
func crossProvider(r *http.Request) string {
	return r.URL.Query().Get("provider")
}

// crossPAN hands out the raw card number.
func crossPAN(c payments.Card) string {
	return c.PAN
}

// crossMasked hands out the safe representation of the same card.
func crossMasked(c payments.Card) string {
	return c.Masked()
}

// crossErrText renders an error, stack and all.
func crossErrText(err error) string {
	return err.Error()
}

// crossSafeText renders a message that carries nothing from the error.
func crossSafeText(err error) string {
	_ = err
	return "Что-то пошло не так. Попробуйте ещё раз."
}
