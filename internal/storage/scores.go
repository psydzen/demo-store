package storage

import (
	"context"
	"fmt"

	"github.com/spndxyz/quiz/internal/domain"
)

// Leaderboard returns the standings of one quiz, best score first. Answers
// still waiting for review contribute no points but are counted separately.
func (db *DB) Leaderboard(ctx context.Context, quizID int64) ([]domain.ScoreRow, error) {
	const query = `
		SELECT u.name,
		       COALESCE(SUM(resp.points_awarded), 0)::int AS points,
		       COUNT(resp.id) FILTER (WHERE resp.is_correct IS NOT NULL)::int AS answered,
		       COUNT(resp.id) FILTER (WHERE resp.is_correct IS NULL)::int AS pending
		FROM attempts a
		JOIN users u ON u.id = a.user_id
		LEFT JOIN responses resp ON resp.attempt_id = a.id
		WHERE a.quiz_id = $1
		GROUP BY u.id, u.name
		ORDER BY points DESC, u.name`

	rows, err := db.pool.Query(ctx, query, quizID)
	if err != nil {
		return nil, fmt.Errorf("leaderboard: %w", err)
	}
	defer rows.Close()

	var out []domain.ScoreRow
	for rows.Next() {
		var s domain.ScoreRow
		if err := rows.Scan(&s.PlayerName, &s.Points, &s.Answered, &s.Pending); err != nil {
			return nil, fmt.Errorf("leaderboard: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("leaderboard: %w", err)
	}
	return out, nil
}

// AttemptScore sums the points of one attempt and counts its answers.
func (db *DB) AttemptScore(ctx context.Context, attemptID int64) (points, answered, pending int, err error) {
	const query = `
		SELECT COALESCE(SUM(points_awarded), 0)::int,
		       COUNT(*) FILTER (WHERE is_correct IS NOT NULL)::int,
		       COUNT(*) FILTER (WHERE is_correct IS NULL)::int
		FROM responses
		WHERE attempt_id = $1`

	if err := db.pool.QueryRow(ctx, query, attemptID).Scan(&points, &answered, &pending); err != nil {
		return 0, 0, 0, fmt.Errorf("attempt score: %w", err)
	}
	return points, answered, pending, nil
}

// AttemptAnswers returns the player's answers in the order they were asked,
// with the verdict and the points each one earned.
func (db *DB) AttemptAnswers(ctx context.Context, attemptID int64) ([]domain.AnswerReview, error) {
	const query = `
		SELECT r.title, q.text, q.kind,
		       COALESCE(o.text, resp.free_text),
		       resp.is_correct, resp.points_awarded
		FROM responses resp
		JOIN questions q ON q.id = resp.question_id
		JOIN rounds r ON r.id = q.round_id
		LEFT JOIN answer_options o ON o.id = resp.option_id
		WHERE resp.attempt_id = $1
		ORDER BY r.position, r.id, q.position, q.id`

	rows, err := db.pool.Query(ctx, query, attemptID)
	if err != nil {
		return nil, fmt.Errorf("attempt answers: %w", err)
	}
	defer rows.Close()

	var out []domain.AnswerReview
	for rows.Next() {
		var a domain.AnswerReview
		err := rows.Scan(&a.RoundTitle, &a.QuestionText, &a.Kind, &a.Answer, &a.IsCorrect, &a.Points)
		if err != nil {
			return nil, fmt.Errorf("attempt answers: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attempt answers: %w", err)
	}
	return out, nil
}

// PendingReviews lists the free-text answers waiting for an admin decision,
// oldest first, together with the hint the quiz author left.
func (db *DB) PendingReviews(ctx context.Context) ([]domain.PendingReview, error) {
	const query = `
		SELECT resp.id, z.title, r.title, q.text, q.admin_hint, resp.free_text, u.name, resp.answered_at
		FROM responses resp
		JOIN questions q ON q.id = resp.question_id
		JOIN rounds r ON r.id = q.round_id
		JOIN quizzes z ON z.id = r.quiz_id
		JOIN attempts a ON a.id = resp.attempt_id
		JOIN users u ON u.id = a.user_id
		WHERE resp.is_correct IS NULL
		ORDER BY resp.answered_at, resp.id`

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("pending reviews: %w", err)
	}
	defer rows.Close()

	var out []domain.PendingReview
	for rows.Next() {
		var p domain.PendingReview
		err := rows.Scan(&p.ResponseID, &p.QuizTitle, &p.RoundTitle, &p.QuestionText,
			&p.AdminHint, &p.FreeText, &p.PlayerName, &p.AnsweredAt)
		if err != nil {
			return nil, fmt.Errorf("pending reviews: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pending reviews: %w", err)
	}
	return out, nil
}

// QuestionForResponse returns the question a response answers, so the caller
// can work out the points to award.
func (db *DB) QuestionForResponse(ctx context.Context, responseID int64) (domain.Question, error) {
	const query = `
		SELECT q.id, q.round_id, q.position, q.kind, q.text,
		       q.points_correct, q.points_wrong, q.admin_hint
		FROM responses resp
		JOIN questions q ON q.id = resp.question_id
		WHERE resp.id = $1`

	q, err := scanQuestion(db.pool.QueryRow(ctx, query, responseID))
	if err != nil {
		return domain.Question{}, fmt.Errorf("question for response: %w", err)
	}
	return q, nil
}

// ReviewResponse records an admin decision on a free-text answer. It only
// touches answers that are still pending, so a double click cannot re-score.
func (db *DB) ReviewResponse(ctx context.Context, responseID int64, correct bool, points int, adminID int64) error {
	const query = `
		UPDATE responses
		SET is_correct = $2, points_awarded = $3, reviewed_at = now(), reviewed_by = $4
		WHERE id = $1 AND is_correct IS NULL`

	tag, err := db.pool.Exec(ctx, query, responseID, correct, points, adminID)
	if err != nil {
		return fmt.Errorf("review response: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("review response: %w", ErrNotFound)
	}
	return nil
}

// PendingCount returns how many answers are waiting for review.
func (db *DB) PendingCount(ctx context.Context) (int, error) {
	var n int
	err := db.pool.QueryRow(ctx, `SELECT count(*)::int FROM responses WHERE is_correct IS NULL`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("pending count: %w", err)
	}
	return n, nil
}
