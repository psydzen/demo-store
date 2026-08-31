package storage

import (
	"context"
	"fmt"

	"github.com/spndxyz/quiz/internal/domain"
)

const userColumns = `id, name, token, role, created_at`

// CreateUser inserts a user and returns the stored row.
func (db *DB) CreateUser(ctx context.Context, name, token string, role domain.Role) (domain.User, error) {
	const query = `
		INSERT INTO users (name, token, role)
		VALUES ($1, $2, $3)
		RETURNING ` + userColumns

	row := db.pool.QueryRow(ctx, query, name, token, role)
	u, err := scanUser(row)
	if err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// UserByToken looks a user up by their login token.
func (db *DB) UserByToken(ctx context.Context, token string) (domain.User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE token = $1`

	u, err := scanUser(db.pool.QueryRow(ctx, query, token))
	if err != nil {
		return domain.User{}, fmt.Errorf("user by token: %w", err)
	}
	return u, nil
}

// UserByID looks a user up by their identifier.
func (db *DB) UserByID(ctx context.Context, id int64) (domain.User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE id = $1`

	u, err := scanUser(db.pool.QueryRow(ctx, query, id))
	if err != nil {
		return domain.User{}, fmt.Errorf("user by id: %w", err)
	}
	return u, nil
}

// ListPlayers returns every player, newest first.
func (db *DB) ListPlayers(ctx context.Context) ([]domain.User, error) {
	const query = `
		SELECT ` + userColumns + `
		FROM users
		WHERE role = 'player'
		ORDER BY created_at DESC, id DESC`

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list players: %w", err)
	}
	defer rows.Close()

	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("list players: %w", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list players: %w", err)
	}
	return out, nil
}

// SetAdminToken replaces the token of the seeded admin. It is how the
// ADMIN_TOKEN environment variable takes effect.
func (db *DB) SetAdminToken(ctx context.Context, token string) error {
	const query = `
		UPDATE users
		SET token = $1
		WHERE id = (SELECT id FROM users WHERE role = 'admin' ORDER BY id LIMIT 1)`

	if _, err := db.pool.Exec(ctx, query, token); err != nil {
		return fmt.Errorf("set admin token: %w", err)
	}
	return nil
}

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanUser(s scanner) (domain.User, error) {
	var u domain.User
	if err := s.Scan(&u.ID, &u.Name, &u.Token, &u.Role, &u.CreatedAt); err != nil {
		return domain.User{}, wrapNotFound(err)
	}
	return u, nil
}
