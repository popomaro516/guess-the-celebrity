package attempt

import (
	"context"

	"github.com/tomy/guess-the-celebrity/services/api/internal/module/image"
	"github.com/tomy/guess-the-celebrity/services/api/internal/module/quiz"
)

type Repository interface {
	Save(ctx context.Context, a Attempt) error
}

type QuizRepository interface {
	FindByID(ctx context.Context, id string) (quiz.Quiz, error)
}

type ImageRepository interface {
	FindByID(ctx context.Context, id string) (image.Image, error)
}
