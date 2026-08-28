package storage

import (
	"context"
	"fmt"

	"github.com/spndxyz/quiz/internal/domain"
)

const quizColumns = `id, title, description, status, created_at`

// CreateQuiz inserts a quiz in draft status.
func (db *DB) CreateQuiz(ctx context.Context, title, description string) (domain.Quiz, error) {
	const query = `
		INSERT INTO quizzes (title, description)
		VALUES ($1, $2)
		RETURNING ` + quizColumns

	q, err := scanQuiz(db.pool.QueryRow(ctx, query, title, description))
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("create quiz: %w", err)
	}
	return q, nil
}

// ListQuizzes returns every quiz, newest first. Admin view.
func (db *DB) ListQuizzes(ctx context.Context) ([]domain.Quiz, error) {
	const query = `SELECT ` + quizColumns + ` FROM quizzes ORDER BY created_at DESC, id DESC`
	return db.queryQuizzes(ctx, query)
}

// ListPublishedQuizzes returns the quizzes players are allowed to enter.
func (db *DB) ListPublishedQuizzes(ctx context.Context) ([]domain.Quiz, error) {
	const query = `
		SELECT ` + quizColumns + `
		FROM quizzes
		WHERE status = 'published'
		ORDER BY created_at DESC, id DESC`
	return db.queryQuizzes(ctx, query)
}

// QuizByID returns a single quiz.
func (db *DB) QuizByID(ctx context.Context, id int64) (domain.Quiz, error) {
	const query = `SELECT ` + quizColumns + ` FROM quizzes WHERE id = $1`

	q, err := scanQuiz(db.pool.QueryRow(ctx, query, id))
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("quiz by id: %w", err)
	}
	return q, nil
}

// SetQuizStatus moves a quiz between draft, published and closed.
func (db *DB) SetQuizStatus(ctx context.Context, id int64, status domain.QuizStatus) error {
	const query = `UPDATE quizzes SET status = $2 WHERE id = $1`

	tag, err := db.pool.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("set quiz status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("set quiz status: %w", ErrNotFound)
	}
	return nil
}

// DeleteQuiz removes a quiz together with its rounds, questions and attempts.
func (db *DB) DeleteQuiz(ctx context.Context, id int64) error {
	if _, err := db.pool.Exec(ctx, `DELETE FROM quizzes WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete quiz: %w", err)
	}
	return nil
}

// CreateRound appends a round to the end of a quiz.
func (db *DB) CreateRound(ctx context.Context, quizID int64, title string) (domain.Round, error) {
	const query = `
		INSERT INTO rounds (quiz_id, title, position)
		VALUES ($1, $2, (SELECT COALESCE(MAX(position), 0) + 1 FROM rounds WHERE quiz_id = $1))
		RETURNING id, quiz_id, title, position`

	var r domain.Round
	err := db.pool.QueryRow(ctx, query, quizID, title).Scan(&r.ID, &r.QuizID, &r.Title, &r.Position)
	if err != nil {
		return domain.Round{}, fmt.Errorf("create round: %w", wrapNotFound(err))
	}
	return r, nil
}

// DeleteRound removes a round and its questions.
func (db *DB) DeleteRound(ctx context.Context, id int64) error {
	if _, err := db.pool.Exec(ctx, `DELETE FROM rounds WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete round: %w", err)
	}
	return nil
}

// RoundQuizID returns the quiz a round belongs to.
func (db *DB) RoundQuizID(ctx context.Context, roundID int64) (int64, error) {
	var quizID int64
	err := db.pool.QueryRow(ctx, `SELECT quiz_id FROM rounds WHERE id = $1`, roundID).Scan(&quizID)
	if err != nil {
		return 0, fmt.Errorf("round quiz id: %w", wrapNotFound(err))
	}
	return quizID, nil
}

// CreateQuestion appends a question to the end of a round.
func (db *DB) CreateQuestion(ctx context.Context, q domain.Question) (domain.Question, error) {
	const query = `
		INSERT INTO questions (round_id, position, kind, text, points_correct, points_wrong, admin_hint)
		VALUES ($1,
		        (SELECT COALESCE(MAX(position), 0) + 1 FROM questions WHERE round_id = $1),
		        $2, $3, $4, $5, $6)
		RETURNING id, round_id, position, kind, text, points_correct, points_wrong, admin_hint`

	row := db.pool.QueryRow(ctx, query, q.RoundID, q.Kind, q.Text, q.PointsCorrect, q.PointsWrong, q.AdminHint)
	out, err := scanQuestion(row)
	if err != nil {
		return domain.Question{}, fmt.Errorf("create question: %w", err)
	}
	return out, nil
}

// DeleteQuestion removes a question and its options.
func (db *DB) DeleteQuestion(ctx context.Context, id int64) error {
	if _, err := db.pool.Exec(ctx, `DELETE FROM questions WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete question: %w", err)
	}
	return nil
}

// QuestionQuizID returns the quiz a question belongs to.
func (db *DB) QuestionQuizID(ctx context.Context, questionID int64) (int64, error) {
	const query = `
		SELECT r.quiz_id
		FROM questions q
		JOIN rounds r ON r.id = q.round_id
		WHERE q.id = $1`

	var quizID int64
	if err := db.pool.QueryRow(ctx, query, questionID).Scan(&quizID); err != nil {
		return 0, fmt.Errorf("question quiz id: %w", wrapNotFound(err))
	}
	return quizID, nil
}

// CreateOption appends an answer option to a question.
func (db *DB) CreateOption(ctx context.Context, questionID int64, text string, isCorrect bool) error {
	const query = `
		INSERT INTO answer_options (question_id, text, is_correct, position)
		VALUES ($1, $2, $3,
		        (SELECT COALESCE(MAX(position), 0) + 1 FROM answer_options WHERE question_id = $1))`

	if _, err := db.pool.Exec(ctx, query, questionID, text, isCorrect); err != nil {
		return fmt.Errorf("create option: %w", err)
	}
	return nil
}

// DeleteOption removes a single answer option.
func (db *DB) DeleteOption(ctx context.Context, id int64) error {
	if _, err := db.pool.Exec(ctx, `DELETE FROM answer_options WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete option: %w", err)
	}
	return nil
}

// OptionQuestionID returns the question an option belongs to.
func (db *DB) OptionQuestionID(ctx context.Context, optionID int64) (int64, error) {
	var questionID int64
	err := db.pool.QueryRow(ctx, `SELECT question_id FROM answer_options WHERE id = $1`, optionID).Scan(&questionID)
	if err != nil {
		return 0, fmt.Errorf("option question id: %w", wrapNotFound(err))
	}
	return questionID, nil
}

// RoundWithQuestions is a round together with everything inside it.
type RoundWithQuestions struct {
	Round     domain.Round
	Questions []domain.Question
}

// QuizStructure loads the whole quiz tree: rounds, their questions and the
// answer options of each question, all in stored order.
func (db *DB) QuizStructure(ctx context.Context, quizID int64) ([]RoundWithQuestions, error) {
	const roundsQuery = `
		SELECT id, quiz_id, title, position
		FROM rounds
		WHERE quiz_id = $1
		ORDER BY position, id`

	rows, err := db.pool.Query(ctx, roundsQuery, quizID)
	if err != nil {
		return nil, fmt.Errorf("quiz structure: %w", err)
	}
	defer rows.Close()

	var out []RoundWithQuestions
	for rows.Next() {
		var r domain.Round
		if err := rows.Scan(&r.ID, &r.QuizID, &r.Title, &r.Position); err != nil {
			return nil, fmt.Errorf("quiz structure: %w", err)
		}
		out = append(out, RoundWithQuestions{Round: r})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("quiz structure: %w", err)
	}

	for i := range out {
		questions, err := db.questionsByRound(ctx, out[i].Round.ID)
		if err != nil {
			return nil, err
		}
		out[i].Questions = questions
	}
	return out, nil
}

func (db *DB) questionsByRound(ctx context.Context, roundID int64) ([]domain.Question, error) {
	const query = `
		SELECT id, round_id, position, kind, text, points_correct, points_wrong, admin_hint
		FROM questions
		WHERE round_id = $1
		ORDER BY position, id`

	rows, err := db.pool.Query(ctx, query, roundID)
	if err != nil {
		return nil, fmt.Errorf("questions by round: %w", err)
	}
	defer rows.Close()

	var out []domain.Question
	for rows.Next() {
		q, err := scanQuestion(rows)
		if err != nil {
			return nil, fmt.Errorf("questions by round: %w", err)
		}
		out = append(out, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("questions by round: %w", err)
	}

	for i := range out {
		opts, err := db.OptionsByQuestion(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Options = opts
	}
	return out, nil
}

// OptionsByQuestion returns the answer options of a question in stored order.
func (db *DB) OptionsByQuestion(ctx context.Context, questionID int64) ([]domain.AnswerOption, error) {
	const query = `
		SELECT id, question_id, text, is_correct, position
		FROM answer_options
		WHERE question_id = $1
		ORDER BY position, id`

	rows, err := db.pool.Query(ctx, query, questionID)
	if err != nil {
		return nil, fmt.Errorf("options by question: %w", err)
	}
	defer rows.Close()

	var out []domain.AnswerOption
	for rows.Next() {
		var o domain.AnswerOption
		if err := rows.Scan(&o.ID, &o.QuestionID, &o.Text, &o.IsCorrect, &o.Position); err != nil {
			return nil, fmt.Errorf("options by question: %w", err)
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("options by question: %w", err)
	}
	return out, nil
}

func (db *DB) queryQuizzes(ctx context.Context, query string, args ...any) ([]domain.Quiz, error) {
	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query quizzes: %w", err)
	}
	defer rows.Close()

	var out []domain.Quiz
	for rows.Next() {
		q, err := scanQuiz(rows)
		if err != nil {
			return nil, fmt.Errorf("query quizzes: %w", err)
		}
		out = append(out, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query quizzes: %w", err)
	}
	return out, nil
}

func scanQuiz(s scanner) (domain.Quiz, error) {
	var q domain.Quiz
	if err := s.Scan(&q.ID, &q.Title, &q.Description, &q.Status, &q.CreatedAt); err != nil {
		return domain.Quiz{}, wrapNotFound(err)
	}
	return q, nil
}

func scanQuestion(s scanner) (domain.Question, error) {
	var q domain.Question
	err := s.Scan(&q.ID, &q.RoundID, &q.Position, &q.Kind, &q.Text,
		&q.PointsCorrect, &q.PointsWrong, &q.AdminHint)
	if err != nil {
		return domain.Question{}, wrapNotFound(err)
	}
	return q, nil
}
