package web

import (
	"net/http"

	"github.com/spndxyz/quiz/internal/auth"
	"github.com/spndxyz/quiz/internal/domain"
	"github.com/spndxyz/quiz/internal/quiz"
	"github.com/spndxyz/quiz/internal/storage"
)

// handleAdminHome sends the admin to the quiz list.
func (s *Server) handleAdminHome(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin/quizzes", http.StatusSeeOther)
}

type playersPage struct {
	base
	Players []domain.User
}

// handlePlayers lists the players and their login tokens.
func (s *Server) handlePlayers(w http.ResponseWriter, r *http.Request) {
	players, err := s.db.ListPlayers(r.Context())
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.mustRender(w, r, http.StatusOK, "admin_players", playersPage{
		base:    s.baseOf(r),
		Players: players,
	})
}

// handleCreatePlayer adds a player with a freshly generated login token.
func (s *Server) handleCreatePlayer(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.badRequest(w, "Не удалось прочитать форму")
		return
	}

	name := formText(r, "name")
	if name == "" {
		s.setFlash(r, "Имя игрока не может быть пустым")
		http.Redirect(w, r, "/admin/players", http.StatusSeeOther)
		return
	}

	token, err := auth.NewToken()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	user, err := s.db.CreateUser(r.Context(), name, token, domain.RolePlayer)
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	s.setFlash(r, "Игрок «"+user.Name+"» создан. Токен: "+user.Token)
	http.Redirect(w, r, "/admin/players", http.StatusSeeOther)
}

type adminQuizzesPage struct {
	base
	Quizzes []domain.Quiz
	Pending int
}

// handleAdminQuizzes lists every quiz, whatever its status.
func (s *Server) handleAdminQuizzes(w http.ResponseWriter, r *http.Request) {
	quizzes, err := s.db.ListQuizzes(r.Context())
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	pending, err := s.db.PendingCount(r.Context())
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.mustRender(w, r, http.StatusOK, "admin_quizzes", adminQuizzesPage{
		base:    s.baseOf(r),
		Quizzes: quizzes,
		Pending: pending,
	})
}

// handleCreateQuiz creates a draft quiz and opens its editor.
func (s *Server) handleCreateQuiz(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.badRequest(w, "Не удалось прочитать форму")
		return
	}

	title := formText(r, "title")
	if title == "" {
		s.setFlash(r, "Название квиза не может быть пустым")
		http.Redirect(w, r, "/admin/quizzes", http.StatusSeeOther)
		return
	}

	q, err := s.db.CreateQuiz(r.Context(), title, formText(r, "description"))
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/admin/quizzes/"+itoa(q.ID), http.StatusSeeOther)
}

// structureView feeds both the editor page and the "quiz_structure" partial
// that htmx swaps in after every edit.
type structureView struct {
	Quiz   domain.Quiz
	Rounds []storage.RoundWithQuestions
}

type editQuizPage struct {
	base
	Structure structureView
}

// handleEditQuiz shows the quiz editor.
func (s *Server) handleEditQuiz(w http.ResponseWriter, r *http.Request) {
	quizID, err := pathInt64(r, "id")
	if err != nil {
		s.notFound(w)
		return
	}

	view, err := s.structure(r, quizID)
	if err != nil {
		s.handleLookupError(w, r, err)
		return
	}
	s.mustRender(w, r, http.StatusOK, "admin_quiz", editQuizPage{
		base:      s.baseOf(r),
		Structure: view,
	})
}

// handleQuizStatus publishes or closes a quiz.
func (s *Server) handleQuizStatus(w http.ResponseWriter, r *http.Request) {
	quizID, err := pathInt64(r, "id")
	if err != nil {
		s.notFound(w)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.badRequest(w, "Не удалось прочитать форму")
		return
	}

	status := domain.QuizStatus(formText(r, "status"))
	switch status {
	case domain.QuizDraft, domain.QuizPublished, domain.QuizClosed:
	default:
		s.badRequest(w, "Неизвестный статус")
		return
	}

	if err := s.db.SetQuizStatus(r.Context(), quizID, status); err != nil {
		s.handleLookupError(w, r, err)
		return
	}
	http.Redirect(w, r, "/admin/quizzes/"+itoa(quizID), http.StatusSeeOther)
}

// handleDeleteQuiz removes a quiz with everything inside it.
func (s *Server) handleDeleteQuiz(w http.ResponseWriter, r *http.Request) {
	quizID, err := pathInt64(r, "id")
	if err != nil {
		s.notFound(w)
		return
	}
	if err := s.db.DeleteQuiz(r.Context(), quizID); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.setFlash(r, "Квиз удалён")
	http.Redirect(w, r, "/admin/quizzes", http.StatusSeeOther)
}

// handleCreateRound appends a round and returns the updated structure.
func (s *Server) handleCreateRound(w http.ResponseWriter, r *http.Request) {
	quizID, err := pathInt64(r, "id")
	if err != nil {
		s.notFound(w)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.badRequest(w, "Не удалось прочитать форму")
		return
	}

	title := formText(r, "title")
	if title == "" {
		title = "Раунд"
	}
	if _, err := s.db.CreateRound(r.Context(), quizID, title); err != nil {
		s.handleLookupError(w, r, err)
		return
	}
	s.renderStructure(w, r, quizID)
}

// handleDeleteRound removes a round and returns the updated structure.
func (s *Server) handleDeleteRound(w http.ResponseWriter, r *http.Request) {
	roundID, err := pathInt64(r, "id")
	if err != nil {
		s.notFound(w)
		return
	}

	quizID, err := s.db.RoundQuizID(r.Context(), roundID)
	if err != nil {
		s.handleLookupError(w, r, err)
		return
	}
	if err := s.db.DeleteRound(r.Context(), roundID); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.renderStructure(w, r, quizID)
}

// handleCreateQuestion appends a question to a round.
func (s *Server) handleCreateQuestion(w http.ResponseWriter, r *http.Request) {
	roundID, err := pathInt64(r, "id")
	if err != nil {
		s.notFound(w)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.badRequest(w, "Не удалось прочитать форму")
		return
	}

	quizID, err := s.db.RoundQuizID(r.Context(), roundID)
	if err != nil {
		s.handleLookupError(w, r, err)
		return
	}

	kind := domain.QuestionKind(formText(r, "kind"))
	if kind != domain.KindChoice && kind != domain.KindFree {
		s.badRequest(w, "Неизвестный тип вопроса")
		return
	}
	text := formText(r, "text")
	if text == "" {
		s.renderStructure(w, r, quizID)
		return
	}

	q := domain.Question{
		RoundID:       roundID,
		Kind:          kind,
		Text:          text,
		PointsCorrect: formIntDefault(r, "points_correct", 1),
		PointsWrong:   formIntDefault(r, "points_wrong", 0),
		AdminHint:     formText(r, "admin_hint"),
	}
	if _, err := s.db.CreateQuestion(r.Context(), q); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.renderStructure(w, r, quizID)
}

// handleDeleteQuestion removes a question.
func (s *Server) handleDeleteQuestion(w http.ResponseWriter, r *http.Request) {
	questionID, err := pathInt64(r, "id")
	if err != nil {
		s.notFound(w)
		return
	}

	quizID, err := s.db.QuestionQuizID(r.Context(), questionID)
	if err != nil {
		s.handleLookupError(w, r, err)
		return
	}
	if err := s.db.DeleteQuestion(r.Context(), questionID); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.renderStructure(w, r, quizID)
}

// handleCreateOption adds an answer option to a choice question.
func (s *Server) handleCreateOption(w http.ResponseWriter, r *http.Request) {
	questionID, err := pathInt64(r, "id")
	if err != nil {
		s.notFound(w)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.badRequest(w, "Не удалось прочитать форму")
		return
	}

	quizID, err := s.db.QuestionQuizID(r.Context(), questionID)
	if err != nil {
		s.handleLookupError(w, r, err)
		return
	}

	if text := formText(r, "text"); text != "" {
		if err := s.db.CreateOption(r.Context(), questionID, text, formBool(r, "is_correct")); err != nil {
			s.serverError(w, r, err)
			return
		}
	}
	s.renderStructure(w, r, quizID)
}

// handleDeleteOption removes a single answer option.
func (s *Server) handleDeleteOption(w http.ResponseWriter, r *http.Request) {
	optionID, err := pathInt64(r, "id")
	if err != nil {
		s.notFound(w)
		return
	}

	questionID, err := s.db.OptionQuestionID(r.Context(), optionID)
	if err != nil {
		s.handleLookupError(w, r, err)
		return
	}
	quizID, err := s.db.QuestionQuizID(r.Context(), questionID)
	if err != nil {
		s.handleLookupError(w, r, err)
		return
	}
	if err := s.db.DeleteOption(r.Context(), optionID); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.renderStructure(w, r, quizID)
}

type reviewPage struct {
	base
	Rows []domain.PendingReview
}

// handleReviewQueue shows the free-text answers waiting for a decision.
func (s *Server) handleReviewQueue(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.PendingReviews(r.Context())
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.mustRender(w, r, http.StatusOK, "admin_review", reviewPage{
		base: s.baseOf(r),
		Rows: rows,
	})
}

// handleReview records the admin's verdict and returns the shortened queue.
func (s *Server) handleReview(w http.ResponseWriter, r *http.Request) {
	responseID, err := pathInt64(r, "id")
	if err != nil {
		s.notFound(w)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.badRequest(w, "Не удалось прочитать форму")
		return
	}

	question, err := s.db.QuestionForResponse(r.Context(), responseID)
	if err != nil {
		s.handleLookupError(w, r, err)
		return
	}

	correct := formBool(r, "correct")
	points := quiz.Points(question, correct)
	admin := userFrom(r)

	// A second click on an already reviewed answer is a no-op, not an error.
	if err := s.db.ReviewResponse(r.Context(), responseID, correct, points, admin.ID); err != nil {
		if !isNotFound(err) {
			s.serverError(w, r, err)
			return
		}
	}

	rows, err := s.db.PendingReviews(r.Context())
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := s.render.fragment(w, "review_list", reviewPage{Rows: rows}); err != nil {
		s.serverError(w, r, err)
	}
}

// structure loads the whole quiz tree for the editor.
func (s *Server) structure(r *http.Request, quizID int64) (structureView, error) {
	q, err := s.db.QuizByID(r.Context(), quizID)
	if err != nil {
		return structureView{}, err
	}
	rounds, err := s.db.QuizStructure(r.Context(), quizID)
	if err != nil {
		return structureView{}, err
	}
	return structureView{Quiz: q, Rounds: rounds}, nil
}

// renderStructure returns the editor body as an htmx fragment.
func (s *Server) renderStructure(w http.ResponseWriter, r *http.Request, quizID int64) {
	view, err := s.structure(r, quizID)
	if err != nil {
		s.handleLookupError(w, r, err)
		return
	}
	if err := s.render.fragment(w, "quiz_structure", view); err != nil {
		s.serverError(w, r, err)
	}
}
