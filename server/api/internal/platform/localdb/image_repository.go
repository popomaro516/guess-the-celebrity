package localdb

import (
	"context"

	"github.com/tomy/guess-the-celebrity/server/api/internal/module/image"
)

const imagesCollection = "images"

type ImageRepository struct {
	store *Store
}

func NewImageRepository(store *Store) *ImageRepository {
	return &ImageRepository{store: store}
}

func (r *ImageRepository) Save(_ context.Context, img image.Image) error {
	r.store.put(imagesCollection, img.ID, img)
	return nil
}

func (r *ImageRepository) FindByID(_ context.Context, id string) (image.Image, error) {
	doc, ok := r.store.get(imagesCollection, id)
	if !ok {
		return image.Image{}, image.ErrImageNotFound
	}
	img, ok := doc.(image.Image)
	if !ok {
		return image.Image{}, image.ErrImageNotFound
	}
	return img, nil
}

func (r *ImageRepository) Update(ctx context.Context, img image.Image) error {
	return r.Save(ctx, img)
}
