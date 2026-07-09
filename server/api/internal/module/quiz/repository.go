package quiz

import (
	"context"

	"github.com/tomy/guess-the-celebrity/server/api/internal/module/image"
)

type Repository interface {
	Save(ctx context.Context, q Quiz) error
	FindByID(ctx context.Context, id string) (Quiz, error)
	FindByCreatorUserID(ctx context.Context, creatorUserID string) ([]Quiz, error)
	Update(ctx context.Context, q Quiz) error
	Delete(ctx context.Context, quizID string) error
}

type PublicFeedRepository interface {
	FindPublicQuizCandidates(ctx context.Context, limit int) ([]PublicQuiz, error)
	Remove(ctx context.Context, quizID string) error
}

type ImageRepository interface {
	FindByID(ctx context.Context, id string) (image.Image, error)
}

type ObjectStore interface {
	Delete(ctx context.Context, objectKey string) error
}
