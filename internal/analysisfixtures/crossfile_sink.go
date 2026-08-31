package analysisfixtures

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/spndxyz/quiz/internal/logtag"
	"github.com/spndxyz/quiz/internal/payments"
)

// The sinks below are reached only through the helpers in crossfile_source.go.
// A rule that reports these is genuinely working across files.

func crossFileSQLBad(ctx context.Context, pool *pgxpool.Pool, r *http.Request) error {
	status := crossStatus(r)
	// ruleid: go-raw-request-to-sql-or-api
	_, err := pool.Query(ctx, "SELECT id FROM payments WHERE status = '"+status+"'")
	return err
}

func crossFileSQLOK(ctx context.Context, pool *pgxpool.Pool, r *http.Request) error {
	status := crossStatus(r)
	// ok: go-raw-request-to-sql-or-api
	_, err := pool.Query(ctx, "SELECT id FROM payments WHERE status = $1", status)
	return err
}

func crossFileAPIBad(r *http.Request) error {
	provider := crossProvider(r)
	// ruleid: go-raw-request-to-sql-or-api
	resp, err := http.Get("https://" + provider + "/receipts")
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func crossFileLogTagBad(ctx context.Context, c payments.Card) {
	// ruleid: go-sensitive-data-in-log-tags
	ctx = logtag.With(ctx, "card", crossPAN(c))
	logtag.From(ctx).Info("charging card")
}

func crossFileLogTagOK(ctx context.Context, c payments.Card) {
	// ok: go-sensitive-data-in-log-tags
	ctx = logtag.With(ctx, "card", crossMasked(c))
	logtag.From(ctx).Info("charging card")
}

func crossFileStacktraceBad(w http.ResponseWriter, err error) {
	// ruleid: go-stacktrace-in-response
	http.Error(w, crossErrText(err), http.StatusInternalServerError)
}

func crossFileStacktraceOK(w http.ResponseWriter, err error) {
	// ok: go-stacktrace-in-response
	http.Error(w, crossSafeText(err), http.StatusInternalServerError)
}
