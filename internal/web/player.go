package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/spndxyz/quiz/internal/domain"
	"github.com/spndxyz/quiz/internal/quiz"
	"github.com/spndxyz/quiz/internal/storage"
)

type loginPage struct {
	base
	Error string
}

// handleLoginForm shows the token form, or sends an already logged-in visitor
// on to their start page.
func (s *Server) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	if id := s.sessions.GetInt64(r.Context(), sessionUserID); id != 0 {
		user, err := s.db.UserByID(r.Context(), id)
		if err == nil {
			http.Redirect(w, r, startPageFor(user), http.StatusSeeOther)
			return
		}
	}
	s.mustRender(w, r, http.StatusOK, "login", loginPage{base: s.baseOf(r)})
}

// handleLogin exchanges a login token for a session.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.badRequest(w, "Не удалось прочитать форму")
		return
	}

	token := strings.TrimSpace(r.PostFormValue("token"))
	if token == "" {
		s.mustRender(w, r, http.StatusUnprocessableEntity, "login",
			loginPage{base: s.baseOf(r), Error: "Введите токен"})
		return
	}

	user, err := s.db.UserByToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			s.mustRender(w, r, http.StatusUnprocessableEntity, "login",
				loginPage{base: s.baseOf(r), Error: "Такой токен не найден"})
			return
		}
		s.serverError(w, r, err)
		return
	}

	// A fresh session token on login closes the session fixation hole.
	if err := s.sessions.RenewToken(r.Context()); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.sessions.Put(r.Context(), sessionUserID, user.ID)
	http.Redirect(w, r, startPageFor(user), http.StatusSeeOther)
}

// handleLogout drops the session.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if err := s.sessions.Destroy(r.Context()); err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func startPageFor(u domain.User) string {
	if u.IsAdmin() {
		return "/admin/quizzes"
	}
	return "/quizzes"
}

type quizCard struct {
	Quiz    domain.Quiz
	Attempt *domain.Attempt
	// Started reports whether the player already has an attempt.
	Started bool
	// Finished reports whether that attempt is complete.
	Finished bool
}

type quizzesPage struct {
	base
	// Active holds the quizzes the player can still play, Finished the ones
	// already completed. They are shown as two columns side by side.
	Active   []quizCard
	Finished []quizCard
}

// handleQuizzes lists the quizzes open to the player.
func (s *Server) handleQuizzes(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)

	quizzes, err := s.db.ListPublishedQuizzes(r.Context())
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	attempts, err := s.db.AttemptsByUser(r.Context(), user.ID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	page := quizzesPage{base: s.baseOf(r)}
	for _, q := range quizzes {
		card := quizCard{Quiz: q}
		if a, ok := attempts[q.ID]; ok {
			attempt := a
			card.Attempt = &attempt
			card.Started = true
			card.Finished = a.FinishedAt != nil
		}
		if card.Finished {
			page.Finished = append(page.Finished, card)
		} else {
			page.Active = append(page.Active, card)
		}
	}

	s.mustRender(w, r, http.StatusOK, "quizzes", page)
}

// handleStartQuiz creates the player's attempt and opens the first question.
func (s *Server) handleStartQuiz(w http.ResponseWriter, r *http.Request) {
	user := userFrom(r)

	quizID, err := pathInt64(r, "id")
	if err != nil {
		s.notFound(w)
		return
	}

	q, err := s.db.QuizByID(r.Context(), quizID)
	if err != nil {
		s.handleLookupError(w, r, err)
		return
	}
	if q.Status != domain.QuizPublished {
		s.badRequest(w, "Этот квиз сейчас недоступен")
		return
	}

	attempt, err := s.db.StartAttempt(r.Context(), quizID, user.ID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, playURL(attempt.ID), http.StatusSeeOther)
}

// questionView feeds the "question" partial. It covers both states the play
// area can be in: a question to answer, or the quiz being over.
type questionView struct {
	AttemptID  int64
	Question   domain.Question
	RoundTitle string
	Number     int
	Total      int
	Finished   bool
	Error      string
}

type playPage struct {
	base
	Quiz     domain.Quiz
	Attempt  domain.Attempt
	Question questionView
}

// handlePlay renders the page holding the current question.
func (s *Server) handlePlay(w http.ResponseWriter, r *http.Request) {
	attempt, ok := s.playerAttempt(w, r)
	if !ok {
		return
	}

	q, err := s.db.QuizByID(r.Context(), attempt.QuizID)
	if err != nil {
		s.handleLookupError(w, r, err)
		return
	}

	view, err := s.nextQuestionView(r, attempt.ID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if view.Finished {
		http.Redirect(w, r, resultsURL(attempt.ID), http.StatusSeeOther)
		return
	}

	s.mustRender(w, r, http.StatusOK, "play", playPage{
		base:     s.baseOf(r),
		Quiz:     q,
		Attempt:  attempt,
		Question: view,
	})
}

// handleAnswer records an answer and swaps in the next question.
func (s *Server) handleAnswer(w http.ResponseWriter, r *http.Request) {
	attempt, ok := s.playerAttempt(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.badRequest(w, "Не удалось прочитать форму")
		return
	}

	questionID, err := formInt64(r, "question_id")
	if err != nil {
		s.badRequest(w, "Некорректный вопрос")
		return
	}

	question, err := s.db.QuestionByID(r.Context(), questionID)
	if err != nil {
		s.handleLookupError(w, r, err)
		return
	}
	// A question from another quiz must never land in this attempt.
	quizID, err := s.db.QuestionQuizID(r.Context(), questionID)
	if err != nil {
		s.handleLookupError(w, r, err)
		return
	}
	if quizID != attempt.QuizID {
		s.notFound(w)
		return
	}

	response, err := s.buildResponse(r, attempt.ID, question)
	if err != nil {
		s.renderQuestionError(w, r, attempt.ID, question, err.Error())
		return
	}
	if err := s.db.SaveResponse(r.Context(), response); err != nil {
		s.serverError(w, r, err)
		return
	}

	view, err := s.nextQuestionView(r, attempt.ID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if view.Finished {
		if err := s.db.FinishAttempt(r.Context(), attempt.ID); err != nil {
			s.serverError(w, r, err)
			return
		}
	}
	if err := s.render.fragment(w, "question", view); err != nil {
		s.serverError(w, r, err)
	}
}

// buildResponse turns the submitted form into a response row. Choice questions
// are scored here and now; free-text answers stay unscored until an admin
// reviews them.
func (s *Server) buildResponse(r *http.Request, attemptID int64, q domain.Question) (domain.Response, error) {
	resp := domain.Response{AttemptID: attemptID, QuestionID: q.ID}

	switch q.Kind {
	case domain.KindChoice:
		optionID, err := formInt64(r, "option_id")
		if err != nil {
			return domain.Response{}, errors.New("Выберите вариант ответа")
		}
		correct, points := quiz.GradeChoice(q, optionID)
		resp.OptionID = &optionID
		resp.IsCorrect = &correct
		resp.PointsAwarded = &points
	case domain.KindFree:
		text := strings.TrimSpace(r.PostFormValue("free_text"))
		if text == "" {
			return domain.Response{}, errors.New("Введите ответ")
		}
		resp.FreeText = text
	default:
		return domain.Response{}, errors.New("Неизвестный тип вопроса")
	}
	return resp, nil
}

// renderQuestionError re-renders the current question with a message.
func (s *Server) renderQuestionError(w http.ResponseWriter, r *http.Request, attemptID int64, q domain.Question, msg string) {
	answered, total, err := s.db.AttemptProgress(r.Context(), attemptID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	view := questionView{
		AttemptID: attemptID,
		Question:  q,
		Number:    answered + 1,
		Total:     total,
		Error:     msg,
	}
	w.WriteHeader(http.StatusUnprocessableEntity)
	if err := s.render.fragment(w, "question", view); err != nil {
		s.serverError(w, r, err)
	}
}

// nextQuestionView loads the next unanswered question, or reports that the
// attempt is over.
func (s *Server) nextQuestionView(r *http.Request, attemptID int64) (questionView, error) {
	pq, err := s.db.NextQuestion(r.Context(), attemptID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return questionView{AttemptID: attemptID, Finished: true}, nil
		}
		return questionView{}, err
	}
	return questionView{
		AttemptID:  attemptID,
		Question:   pq.Question,
		RoundTitle: pq.RoundTitle,
		Number:     pq.Number,
		Total:      pq.Total,
	}, nil
}

// leaderboardView feeds the "leaderboard_table" partial, shared by the results
// page and the standalone leaderboard page.
type leaderboardView struct {
	Rows []domain.ScoreRow
	// Highlight is the name of the player looking at the table, so their own
	// row stands out. Empty for an admin.
	Highlight string
}

type resultsPage struct {
	base
	Quiz        domain.Quiz
	Points      int
	Answered    int
	Pending     int
	Leaderboard leaderboardView
}

// handleResults shows what the player scored.
func (s *Server) handleResults(w http.ResponseWriter, r *http.Request) {
	attempt, ok := s.playerAttempt(w, r)
	if !ok {
		return
	}

	q, err := s.db.QuizByID(r.Context(), attempt.QuizID)
	if err != nil {
		s.handleLookupError(w, r, err)
		return
	}
	points, answered, pending, err := s.db.AttemptScore(r.Context(), attempt.ID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	board, err := s.leaderboardOf(r, attempt.QuizID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	s.mustRender(w, r, http.StatusOK, "results", resultsPage{
		base:        s.baseOf(r),
		Quiz:        q,
		Points:      points,
		Answered:    answered,
		Pending:     pending,
		Leaderboard: board,
	})
}

type leaderboardPage struct {
	base
	Quiz        domain.Quiz
	Leaderboard leaderboardView
}

// handleLeaderboard shows the standings of a quiz.
func (s *Server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	quizID, err := pathInt64(r, "quizID")
	if err != nil {
		s.notFound(w)
		return
	}

	q, err := s.db.QuizByID(r.Context(), quizID)
	if err != nil {
		s.handleLookupError(w, r, err)
		return
	}
	board, err := s.leaderboardOf(r, quizID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	s.mustRender(w, r, http.StatusOK, "leaderboard", leaderboardPage{
		base:        s.baseOf(r),
		Quiz:        q,
		Leaderboard: board,
	})
}

// leaderboardOf loads the standings of a quiz and marks the current player.
func (s *Server) leaderboardOf(r *http.Request, quizID int64) (leaderboardView, error) {
	rows, err := s.db.Leaderboard(r.Context(), quizID)
	if err != nil {
		return leaderboardView{}, err
	}

	view := leaderboardView{Rows: rows}
	if u := userFrom(r); u != nil && !u.IsAdmin() {
		view.Highlight = u.Name
	}
	return view, nil
}

// playerAttempt loads the attempt named in the path and checks it belongs to
// the current player. Admins may look at any attempt.
func (s *Server) playerAttempt(w http.ResponseWriter, r *http.Request) (domain.Attempt, bool) {
	id, err := pathInt64(r, "attemptID")
	if err != nil {
		s.notFound(w)
		return domain.Attempt{}, false
	}

	attempt, err := s.db.AttemptByID(r.Context(), id)
	if err != nil {
		s.handleLookupError(w, r, err)
		return domain.Attempt{}, false
	}

	user := userFrom(r)
	if attempt.UserID != user.ID && !user.IsAdmin() {
		s.notFound(w)
		return domain.Attempt{}, false
	}
	return attempt, true
}

// isNotFound reports whether the error came from a lookup that matched no row.
func isNotFound(err error) bool { return errors.Is(err, storage.ErrNotFound) }

func (s *Server) handleLookupError(w http.ResponseWriter, r *http.Request, err error) {
	if isNotFound(err) {
		s.notFound(w)
		return
	}
	s.serverError(w, r, err)
}

// mustRender renders a page and turns a template failure into a 500.
func (s *Server) mustRender(w http.ResponseWriter, r *http.Request, status int, name string, data any) {
	if err := s.render.page(w, status, name, data); err != nil {
		s.serverError(w, r, err)
	}
}

func playURL(attemptID int64) string    { return "/play/" + itoa(attemptID) }
func resultsURL(attemptID int64) string { return "/results/" + itoa(attemptID) }
