package attempt

import (
	"context"

	"github.com/tomy/guess-the-celebrity/server/api/internal/module/quiz"
)

type Service struct {
	quizzes QuizRepository
	images  ImageRepository
}

func NewService(quizzes QuizRepository, images ImageRepository) *Service {
	return &Service{quizzes: quizzes, images: images}
}

type AnswerInput struct {
	QuizID string
	UserID string
	Answer string
}

type AnswerOutput struct {
	Correct          bool
	CorrectAnswer    string
	OriginalImageKey string
}

func (s *Service) Answer(ctx context.Context, in AnswerInput) (AnswerOutput, error) {
	q, err := s.quizzes.FindByID(ctx, in.QuizID)
	if err != nil {
		return AnswerOutput{}, err
	}
	if q.Status != quiz.StatusPublished {
		return AnswerOutput{}, ErrQuizNotPublished
	}

	correct := in.Answer == q.Answer
	if !correct {
		return AnswerOutput{Correct: false}, nil
	}

	img, err := s.images.FindByID(ctx, q.ImageID)
	if err != nil {
		return AnswerOutput{}, err
	}
	return AnswerOutput{
		Correct:          true,
		CorrectAnswer:    q.Answer,
		OriginalImageKey: img.OriginalImageKey,
	}, nil
}
