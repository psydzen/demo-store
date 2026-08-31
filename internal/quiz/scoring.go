// Package quiz contains the scoring rules.
package quiz

import "github.com/spndxyz/quiz/internal/domain"

// Points returns the score for a question given whether the answer was right.
func Points(q domain.Question, correct bool) int {
	if correct {
		return q.PointsCorrect
	}
	return q.PointsWrong
}

// GradeChoice scores an answer to a multiple-choice question. It reports
// whether the picked option is one of the correct ones and the points earned.
//
// An option that does not belong to the question counts as a wrong answer.
func GradeChoice(q domain.Question, optionID int64) (correct bool, points int) {
	for _, opt := range q.Options {
		if opt.ID == optionID {
			correct = opt.IsCorrect
			break
		}
	}
	return correct, Points(q, correct)
}
