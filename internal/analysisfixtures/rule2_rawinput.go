package analysisfixtures

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func rawInputToSQLBad(ctx context.Context, pool *pgxpool.Pool, r *http.Request) error {
	status := r.URL.Query().Get("status")
	// ruleid: go-raw-request-to-sql-or-api
	_, err := pool.Query(ctx, "SELECT id FROM payments WHERE status = '"+status+"'")
	return err
}

func rawInputToSQLBadExec(ctx context.Context, pool *pgxpool.Pool, r *http.Request) error {
	name := r.FormValue("name")
	// ruleid: go-raw-request-to-sql-or-api
	_, err := pool.Exec(ctx, "UPDATE users SET name = '"+name+"' WHERE id = 1")
	return err
}

func rawInputToSQLOK(ctx context.Context, pool *pgxpool.Pool, r *http.Request) error {
	status := r.URL.Query().Get("status")
	// ok: go-raw-request-to-sql-or-api
	_, err := pool.Query(ctx, "SELECT id FROM payments WHERE status = $1", status)
	return err
}

func rawInputToAPIBad(r *http.Request) error {
	provider := r.URL.Query().Get("provider")
	// ruleid: go-raw-request-to-sql-or-api
	resp, err := http.Get("https://" + provider + "/receipts")
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func rawInputToAPIOK(r *http.Request) error {
	_ = r.URL.Query().Get("provider")
	// ok: go-raw-request-to-sql-or-api
	resp, err := http.Get("https://payments.internal/receipts")
	if err != nil {
		return err
	}
	return resp.Body.Close()
}
