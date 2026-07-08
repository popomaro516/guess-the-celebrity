package image

import "context"

type Repository interface {
	Save(ctx context.Context, img Image) error
	FindByID(ctx context.Context, id string) (Image, error)
	Update(ctx context.Context, img Image) error
}
