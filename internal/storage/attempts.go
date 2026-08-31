package storage

import (
	"context"
	"fmt"

	"github.com/spndxyz/quiz/internal/domain"
)

const attemptColumns = `id, quiz_id, user_id, started_at, finished_at`

// StartAttempt returns the player's attempt at a quiz, creating it on the
// first call. A player gets exactly one attempt per quiz.
func (db *DB) StartAttempt(ctx context.Context, quizID, userID int64) (domain.Attempt, error) {
	// The no-op update lets the statement return the existing row when the
	// player has already started this quiz.
	const query = `
		INSERT INTO attempts (quiz_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (quiz_id, user_id) DO UPDATE SET quiz_id = EXCLUDED.quiz_id
		RETURNING ` + attemptColumns

	a, err := scanAttempt(db.pool.QueryRow(ctx, query, quizID, userID))
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("start attempt: %w", err)
	}
	return a, nil
}

// AttemptByID returns a single attempt.
func (db *DB) AttemptByID(ctx context.Context, id int64) (domain.Attempt, error) {
	const query = `SELECT ` + attemptColumns + ` FROM attempts WHERE id = $1`

	a, err := scanAttempt(db.pool.QueryRow(ctx, query, id))
	if err != nil {
		return domain.Attempt{}, fmt.Errorf("attempt by id: %w", err)
	}
	return a, nil
}

// AttemptsByUser returns the attempts a player has made, keyed by quiz.
func (db *DB) AttemptsByUser(ctx context.Context, userID int64) (map[int64]domain.Attempt, error) {
	const query = `SELECT ` + attemptColumns + ` FROM attempts WHERE user_id = $1`

	rows, err := db.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("attempts by user: %w", err)
	}
	defer rows.Close()

	out := make(map[int64]domain.Attempt)
	for rows.Next() {
		a, err := scanAttempt(rows)
		if err != nil {
			return nil, fmt.Errorf("attempts by user: %w", err)
		}
		out[a.QuizID] = a
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("attempts by user: %w", err)
	}
	return out, nil
}

// FinishAttempt marks an attempt as completed. Calling it twice is harmless.
func (db *DB) FinishAttempt(ctx context.Context, attemptID int64) error {
	const query = `UPDATE attempts SET finished_at = now() WHERE id = $1 AND finished_at IS NULL`

	if _, err := db.pool.Exec(ctx, query, attemptID); err != nil {
		return fmt.Errorf("finish attempt: %w", err)
	}
	return nil
}

// PlayQuestion is the question a player currently has to answer, with enough
// context to render a progress line.
type PlayQuestion struct {
	Question   domain.Question
	RoundTitle string
	// Number is the 1-based position of this question in the whole quiz.
	Number int
	Total  int
}

// NextQuestion returns the first question of the attempt that has no answer
// yet, walking rounds and questions in their stored order. It returns
// ErrNotFound once every question has been answered.
func (db *DB) NextQuestion(ctx context.Context, attemptID int64) (PlayQuestion, error) {
	const query = `
		SELECT q.id, q.round_id, q.position, q.kind, q.text,
		       q.points_correct, q.points_wrong, q.admin_hint, r.title
		FROM attempts a
		JOIN rounds r ON r.quiz_id = a.quiz_id
		JOIN questions q ON q.round_id = r.id
		WHERE a.id = $1
		  AND NOT EXISTS (
		      SELECT 1 FROM responses resp
		      WHERE resp.attempt_id = a.id AND resp.question_id = q.id)
		ORDER BY r.position, r.id, q.position, q.id
		LIMIT 1`

	var pq PlayQuestion
	q := &pq.Question
	err := db.pool.QueryRow(ctx, query, attemptID).Scan(
		&q.ID, &q.RoundID, &q.Position, &q.Kind, &q.Text,
		&q.PointsCorrect, &q.PointsWrong, &q.AdminHint, &pq.RoundTitle)
	if err != nil {
		return PlayQuestion{}, fmt.Errorf("next question: %w", wrapNotFound(err))
	}

	opts, err := db.OptionsByQuestion(ctx, q.ID)
	if err != nil {
		return PlayQuestion{}, err
	}
	q.Options = opts

	answered, total, err := db.AttemptProgress(ctx, attemptID)
	if err != nil {
		return PlayQuestion{}, err
	}
	pq.Number = answered + 1
	pq.Total = total
	return pq, nil
}

// QuestionByID returns a question with its answer options.
func (db *DB) QuestionByID(ctx context.Context, id int64) (domain.Question, error) {
	const query = `
		SELECT id, round_id, position, kind, text, points_correct, points_wrong, admin_hint
		FROM questions
		WHERE id = $1`

	q, err := scanQuestion(db.pool.QueryRow(ctx, query, id))
	if err != nil {
		return domain.Question{}, fmt.Errorf("question by id: %w", err)
	}
	opts, err := db.OptionsByQuestion(ctx, q.ID)
	if err != nil {
		return domain.Question{}, err
	}
	q.Options = opts
	return q, nil
}

// AttemptProgress reports how many questions of the attempt are answered and
// how many the quiz holds in total.
func (db *DB) AttemptProgress(ctx context.Context, attemptID int64) (answered, total int, err error) {
	const query = `
		SELECT
		    (SELECT count(*) FROM responses WHERE attempt_id = a.id),
		    (SELECT count(*)
		     FROM rounds r JOIN questions q ON q.round_id = r.id
		     WHERE r.quiz_id = a.quiz_id)
		FROM attempts a
		WHERE a.id = $1`

	if err := db.pool.QueryRow(ctx, query, attemptID).Scan(&answered, &total); err != nil {
		return 0, 0, fmt.Errorf("attempt progress: %w", wrapNotFound(err))
	}
	return answered, total, nil
}

// SaveResponse stores a player's answer. An answer that is already recorded is
// kept as is, so a resubmitted form cannot overwrite or double-score it.
func (db *DB) SaveResponse(ctx context.Context, r domain.Response) error {
	const query = `
		INSERT INTO responses (attempt_id, question_id, option_id, free_text, is_correct, points_awarded)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (attempt_id, question_id) DO NOTHING`

	_, err := db.pool.Exec(ctx, query,
		r.AttemptID, r.QuestionID, r.OptionID, r.FreeText, r.IsCorrect, r.PointsAwarded)
	if err != nil {
		return fmt.Errorf("save response: %w", err)
	}
	return nil
}

func scanAttempt(s scanner) (domain.Attempt, error) {
	var a domain.Attempt
	if err := s.Scan(&a.ID, &a.QuizID, &a.UserID, &a.StartedAt, &a.FinishedAt); err != nil {
		return domain.Attempt{}, wrapNotFound(err)
	}
	return a, nil
}
