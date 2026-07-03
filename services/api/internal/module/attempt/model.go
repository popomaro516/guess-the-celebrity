package attempt

import (
	"context"
	"errors"
	"time"

	"github.com/tomy/guess-the-celebrity/services/api/internal/module/image"
	"github.com/tomy/guess-the-celebrity/services/api/internal/module/quiz"
)

var ErrQuizNotPublished = errors.New("quiz is not published")

type Attempt struct {
	ID        string
	QuizID    string
	UserID    string
	Answer    string
	IsCorrect bool
	CreatedAt time.Time
}

type Repository interface {
	Save(ctx context.Context, a Attempt) error
}

type QuizRepository interface {
	FindByID(ctx context.Context, id string) (quiz.Quiz, error)
}

type ImageRepository interface {
	FindByID(ctx context.Context, id string) (image.Image, error)
}

type IDGenerator interface {
	NewID(prefix string) string
}

type Clock interface {
	Now() time.Time
}
