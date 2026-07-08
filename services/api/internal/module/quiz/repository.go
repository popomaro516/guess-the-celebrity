package quiz

import (
	"context"

	"github.com/tomy/guess-the-celebrity/services/api/internal/module/image"
)

type Repository interface {
	Save(ctx context.Context, q Quiz) error
	FindByID(ctx context.Context, id string) (Quiz, error)
	FindRandomPublished(ctx context.Context) (Quiz, error)
	Update(ctx context.Context, q Quiz) error
}

type ImageRepository interface {
	FindByID(ctx context.Context, id string) (image.Image, error)
}
