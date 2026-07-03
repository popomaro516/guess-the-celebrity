package attempt

import (
	"context"

	"github.com/tomy/guess-the-celebrity/services/api/internal/module/quiz"
)

type Service struct {
	repo    Repository
	quizzes QuizRepository
	images  ImageRepository
	ids     IDGenerator
	clock   Clock
}

func NewService(repo Repository, quizzes QuizRepository, images ImageRepository, ids IDGenerator, clock Clock) *Service {
	return &Service{repo: repo, quizzes: quizzes, images: images, ids: ids, clock: clock}
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
	if err := s.repo.Save(ctx, Attempt{
		ID:        s.ids.NewID("attempt"),
		QuizID:    in.QuizID,
		UserID:    in.UserID,
		Answer:    in.Answer,
		IsCorrect: correct,
		CreatedAt: s.clock.Now(),
	}); err != nil {
		return AnswerOutput{}, err
	}
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
