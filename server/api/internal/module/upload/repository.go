package upload

import (
	"context"
	"time"

	"github.com/tomy/guess-the-celebrity/server/api/internal/module/image"
)

type ImageRepository interface {
	Save(ctx context.Context, img image.Image) error
	FindByID(ctx context.Context, id string) (image.Image, error)
	Update(ctx context.Context, img image.Image) error
}

type Presigner interface {
	PresignPut(ctx context.Context, objectKey string, contentType string, expiresIn time.Duration) (string, error)
}

type ObjectStore interface {
	Exists(ctx context.Context, objectKey string) (bool, error)
}
