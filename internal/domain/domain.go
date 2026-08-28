// Package domain holds the data structures shared across the application.
package domain

import "time"

// Role identifies what a user is allowed to do.
type Role string

// Supported roles.
const (
	RoleAdmin  Role = "admin"
	RolePlayer Role = "player"
)

// QuizStatus is the lifecycle stage of a quiz.
type QuizStatus string

// Supported quiz statuses. Players only ever see published quizzes.
const (
	QuizDraft     QuizStatus = "draft"
	QuizPublished QuizStatus = "published"
	QuizClosed    QuizStatus = "closed"
)

// QuestionKind tells how a question is answered and scored.
type QuestionKind string

// Supported question kinds.
const (
	// KindChoice is answered by picking one of the stored options and is
	// scored the moment it is submitted.
	KindChoice QuestionKind = "choice"
	// KindFree is answered with free text and is scored by an admin.
	KindFree QuestionKind = "free"
)

// User is either an admin or a player. The token is the only credential.
type User struct {
	ID        int64
	Name      string
	Token     string
	Role      Role
	CreatedAt time.Time
}

// IsAdmin reports whether the user may access the admin pages.
func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

// Quiz is a collection of rounds.
type Quiz struct {
	ID          int64
	Title       string
	Description string
	Status      QuizStatus
	CreatedAt   time.Time
}

// Round groups questions inside a quiz.
type Round struct {
	ID       int64
	QuizID   int64
	Title    string
	Position int
}

// Question is a single scored item inside a round.
type Question struct {
	ID            int64
	RoundID       int64
	Position      int
	Kind          QuestionKind
	Text          string
	PointsCorrect int
	PointsWrong   int
	// AdminHint is shown to the admin while reviewing free-text answers.
	AdminHint string
	Options   []AnswerOption
}

// AnswerOption is one of the choices offered for a KindChoice question.
type AnswerOption struct {
	ID         int64
	QuestionID int64
	Text       string
	IsCorrect  bool
	Position   int
}

// Attempt is one player's run through one quiz.
type Attempt struct {
	ID         int64
	QuizID     int64
	UserID     int64
	StartedAt  time.Time
	FinishedAt *time.Time
}

// Response is a player's answer to a single question.
//
// IsCorrect and PointsAwarded are nil while a free-text answer waits for an
// admin decision; that is exactly what puts it in the review queue.
type Response struct {
	ID            int64
	AttemptID     int64
	QuestionID    int64
	OptionID      *int64
	FreeText      string
	IsCorrect     *bool
	PointsAwarded *int
	AnsweredAt    time.Time
	ReviewedAt    *time.Time
	ReviewedBy    *int64
}

// Pending reports whether the response still needs an admin decision.
func (r Response) Pending() bool { return r.IsCorrect == nil }

// PendingReview is one row of the admin review queue.
type PendingReview struct {
	ResponseID   int64
	QuizTitle    string
	RoundTitle   string
	QuestionText string
	AdminHint    string
	FreeText     string
	PlayerName   string
	AnsweredAt   time.Time
}

// ScoreRow is one line of a quiz leaderboard.
type ScoreRow struct {
	PlayerName string
	Points     int
	Answered   int
	Pending    int
}

// AttemptResult summarises a finished attempt for the player.
type AttemptResult struct {
	Quiz     Quiz
	Points   int
	Answered int
	Pending  int
}
