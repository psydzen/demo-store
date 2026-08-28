package quiz_test

import (
	"testing"

	"github.com/spndxyz/quiz/internal/domain"
	"github.com/spndxyz/quiz/internal/quiz"
)

func question() domain.Question {
	return domain.Question{
		ID:            7,
		Kind:          domain.KindChoice,
		PointsCorrect: 10,
		PointsWrong:   -3,
		Options: []domain.AnswerOption{
			{ID: 1, Text: "wrong", IsCorrect: false},
			{ID: 2, Text: "right", IsCorrect: true},
		},
	}
}

func TestGradeChoice(t *testing.T) {
	tests := []struct {
		name        string
		optionID    int64
		wantCorrect bool
		wantPoints  int
	}{
		{name: "correct option", optionID: 2, wantCorrect: true, wantPoints: 10},
		{name: "wrong option", optionID: 1, wantCorrect: false, wantPoints: -3},
		{name: "option from another question", optionID: 99, wantCorrect: false, wantPoints: -3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			correct, points := quiz.GradeChoice(question(), tt.optionID)
			if correct != tt.wantCorrect {
				t.Errorf("GradeChoice(%d) correct = %v, want %v", tt.optionID, correct, tt.wantCorrect)
			}
			if points != tt.wantPoints {
				t.Errorf("GradeChoice(%d) points = %d, want %d", tt.optionID, points, tt.wantPoints)
			}
		})
	}
}

func TestPoints(t *testing.T) {
	q := question()
	if got, want := quiz.Points(q, true), 10; got != want {
		t.Errorf("Points(correct) = %d, want %d", got, want)
	}
	if got, want := quiz.Points(q, false), -3; got != want {
		t.Errorf("Points(wrong) = %d, want %d", got, want)
	}
}
