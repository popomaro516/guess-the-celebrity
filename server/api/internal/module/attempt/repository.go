package attempt

import (
	"context"

	"github.com/tomy/guess-the-celebrity/server/api/internal/module/image"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/quiz"
)

type QuizRepository interface {
	FindByID(ctx context.Context, id string) (quiz.Quiz, error)
}

type ImageRepository interface {
	FindByID(ctx context.Context, id string) (image.Image, error)
}
