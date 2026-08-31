// Command quiz runs the quiz web application.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"google.golang.org/grpc"

	"github.com/spndxyz/quiz/internal/config"
	"github.com/spndxyz/quiz/internal/grpcapi"
	"github.com/spndxyz/quiz/internal/storage"
	"github.com/spndxyz/quiz/internal/web"
	"github.com/spndxyz/quiz/migrations"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(log); err != nil {
		log.Error("startup failed", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := applyMigrations(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	log.Info("migrations applied")

	db, err := storage.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	if cfg.AdminToken != "" {
		if err := db.SetAdminToken(ctx, cfg.AdminToken); err != nil {
			return fmt.Errorf("apply admin token: %w", err)
		}
		log.Info("admin token taken from ADMIN_TOKEN")
	}

	sessions := scs.New()
	sessions.Store = pgxstore.New(db.Pool())
	sessions.Lifetime = cfg.SessionLifetime
	sessions.Cookie.HttpOnly = true
	sessions.Cookie.SameSite = http.SameSiteLaxMode

	server, err := web.New(db, sessions, log)
	if err != nil {
		return fmt.Errorf("build server: %w", err)
	}
	handler, err := server.Handler()
	if err != nil {
		return fmt.Errorf("build handler: %w", err)
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	public := grpcapi.NewPublicServer(db, log)
	admin := grpcapi.NewAdminServer(db)

	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()
	go serveGRPC(log, public, cfg.GRPCAddr, "public", errc)
	go serveGRPC(log, admin, cfg.GRPCAdminAddr, "admin", errc)

	select {
	case err := <-errc:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
		log.Info("shutting down")
	}

	public.GracefulStop()
	admin.GracefulStop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

// serveGRPC listens on addr and blocks until the server stops.
func serveGRPC(log *slog.Logger, srv *grpc.Server, addr, name string, errc chan<- error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		errc <- fmt.Errorf("listen %s grpc: %w", name, err)
		return
	}
	log.Info("grpc listening", "name", name, "addr", addr)
	if err := srv.Serve(lis); err != nil {
		errc <- fmt.Errorf("serve %s grpc: %w", name, err)
	}
}

// applyMigrations brings the schema up to date using the embedded SQL files.
func applyMigrations(databaseURL string) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("open migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, migrateURL(databaseURL))
	if err != nil {
		return fmt.Errorf("open migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// migrateURL rewrites the connection string to the scheme golang-migrate's
// pgx driver registers itself under.
func migrateURL(databaseURL string) string {
	for _, prefix := range []string{"postgres://", "postgresql://"} {
		if strings.HasPrefix(databaseURL, prefix) {
			return "pgx5://" + strings.TrimPrefix(databaseURL, prefix)
		}
	}
	return databaseURL
}
