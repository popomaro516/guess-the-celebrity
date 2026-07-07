package upload

import (
	"context"
	"errors"
	"time"

	"github.com/tomy/guess-the-celebrity/services/api/internal/module/image"
)

var (
	ErrInvalidUpload        = errors.New("invalid upload")
	ErrUploadObjectNotFound = errors.New("upload object not found")
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

type IDGenerator interface {
	NewID(prefix string) string
}

type Clock interface {
	Now() time.Time
}
