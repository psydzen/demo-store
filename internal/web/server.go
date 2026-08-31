// Package web serves the HTML interface for players and admins.
package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/alexedwards/scs/v2"

	"github.com/spndxyz/quiz/internal/domain"
	"github.com/spndxyz/quiz/internal/storage"
)

// Session keys.
const (
	sessionUserID = "userID"
	sessionFlash  = "flash"
)

// contextKey keeps the request-scoped values out of the string namespace.
type contextKey struct{ name string }

var userContextKey = &contextKey{"user"}

// Server wires the storage, the session manager and the templates into an
// http.Handler.
type Server struct {
	db       *storage.DB
	sessions *scs.SessionManager
	log      *slog.Logger
	render   *renderer
}

// New builds a Server and parses the templates.
func New(db *storage.DB, sessions *scs.SessionManager, log *slog.Logger) (*Server, error) {
	r, err := newRenderer()
	if err != nil {
		return nil, fmt.Errorf("new renderer: %w", err)
	}
	return &Server{db: db, sessions: sessions, log: log, render: r}, nil
}

// Handler returns the router with the session middleware already applied.
func (s *Server) Handler() (http.Handler, error) {
	static, err := staticFS()
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))

	// Public.
	mux.HandleFunc("GET /{$}", s.handleLoginForm)
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("POST /logout", s.handleLogout)

	// Player.
	player := func(h http.HandlerFunc) http.Handler { return s.requireUser(h) }
	mux.Handle("GET /quizzes", player(s.handleQuizzes))
	mux.Handle("POST /quizzes/{id}/start", player(s.handleStartQuiz))
	mux.Handle("GET /play/{attemptID}", player(s.handlePlay))
	mux.Handle("POST /play/{attemptID}/answer", player(s.handleAnswer))
	mux.Handle("GET /results/{attemptID}", player(s.handleResults))
	mux.Handle("GET /leaderboard/{quizID}", player(s.handleLeaderboard))

	// Admin.
	admin := func(h http.HandlerFunc) http.Handler { return s.requireAdmin(h) }
	mux.Handle("GET /admin", admin(s.handleAdminHome))
	mux.Handle("GET /admin/players", admin(s.handlePlayers))
	mux.Handle("POST /admin/players", admin(s.handleCreatePlayer))
	mux.Handle("GET /admin/quizzes", admin(s.handleAdminQuizzes))
	mux.Handle("POST /admin/quizzes", admin(s.handleCreateQuiz))
	mux.Handle("GET /admin/quizzes/{id}", admin(s.handleEditQuiz))
	mux.Handle("POST /admin/quizzes/{id}/status", admin(s.handleQuizStatus))
	mux.Handle("POST /admin/quizzes/{id}/delete", admin(s.handleDeleteQuiz))
	mux.Handle("POST /admin/quizzes/{id}/rounds", admin(s.handleCreateRound))
	mux.Handle("POST /admin/rounds/{id}/delete", admin(s.handleDeleteRound))
	mux.Handle("POST /admin/rounds/{id}/questions", admin(s.handleCreateQuestion))
	mux.Handle("POST /admin/questions/{id}/delete", admin(s.handleDeleteQuestion))
	mux.Handle("POST /admin/questions/{id}/options", admin(s.handleCreateOption))
	mux.Handle("POST /admin/options/{id}/delete", admin(s.handleDeleteOption))
	mux.Handle("GET /admin/payments", admin(s.handlePaymentSearch))
	mux.Handle("GET /admin/payments/receipt", admin(s.handlePaymentReceipt))
	mux.Handle("GET /admin/payers", admin(s.handlePayerProfile))
	mux.Handle("GET /admin/review", admin(s.handleReviewQueue))
	mux.Handle("POST /admin/review/{id}", admin(s.handleReview))

	return s.sessions.LoadAndSave(mux), nil
}

// requireUser rejects visitors without a valid session and puts the user into
// the request context.
func (s *Server) requireUser(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := s.sessions.GetInt64(r.Context(), sessionUserID)
		if id == 0 {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		user, err := s.db.UserByID(r.Context(), id)
		if err != nil {
			// The account is gone; drop the stale session.
			if errors.Is(err, storage.ErrNotFound) {
				_ = s.sessions.Destroy(r.Context())
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			s.serverError(w, r, err)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireAdmin additionally checks the role.
func (s *Server) requireAdmin(next http.HandlerFunc) http.Handler {
	return s.requireUser(func(w http.ResponseWriter, r *http.Request) {
		if u := userFrom(r); u == nil || !u.IsAdmin() {
			http.Error(w, "Доступ только для администратора", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// userFrom returns the user the middleware stored, or nil.
func userFrom(r *http.Request) *domain.User {
	u, _ := r.Context().Value(userContextKey).(*domain.User)
	return u
}

// baseOf builds the fields shared by every page and consumes the flash message.
func (s *Server) baseOf(r *http.Request) base {
	return base{
		User:  userFrom(r),
		Flash: s.sessions.PopString(r.Context(), sessionFlash),
	}
}

// setFlash queues a one-off message for the next page the user opens.
func (s *Server) setFlash(r *http.Request, msg string) {
	s.sessions.Put(r.Context(), sessionFlash, msg)
}

func (s *Server) serverError(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	http.Error(w, "Что-то пошло не так. Попробуйте ещё раз.", http.StatusInternalServerError)
}

func (s *Server) notFound(w http.ResponseWriter) {
	http.Error(w, "Страница не найдена", http.StatusNotFound)
}

func (s *Server) badRequest(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusBadRequest)
}

// pathInt64 reads a numeric path segment.
func pathInt64(r *http.Request, name string) (int64, error) {
	v, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse path value %s: %w", name, err)
	}
	return v, nil
}
