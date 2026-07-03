package upload_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tomy/guess-the-celebrity/services/api/internal/module/image"
	"github.com/tomy/guess-the-celebrity/services/api/internal/module/upload"
)

func TestPresignCreatesPendingImage(t *testing.T) {
	images := newFakeImageRepository()
	presigner := fakePresigner{}
	svc := upload.NewService(images, presigner, fixedIDs{"img_123"}, fixedClock{})

	got, err := svc.Presign(context.Background(), upload.PresignInput{
		Filename:    "cat.jpg",
		ContentType: "image/jpeg",
		Size:        2048,
	})
	if err != nil {
		t.Fatalf("Presign returned error: %v", err)
	}
	if got.ImageID != "img_123" {
		t.Fatalf("ImageID = %q, want img_123", got.ImageID)
	}
	if got.ObjectKey != "originals/anonymous/img_123/source.jpg" {
		t.Fatalf("ObjectKey = %q", got.ObjectKey)
	}
	if got.ExpiresIn != 300 {
		t.Fatalf("ExpiresIn = %d, want 300", got.ExpiresIn)
	}

	saved, err := images.FindByID(context.Background(), "img_123")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if saved.Status != image.StatusPendingUpload {
		t.Fatalf("Status = %q, want %q", saved.Status, image.StatusPendingUpload)
	}
}

func TestPresignRejectsUnsupportedContentType(t *testing.T) {
	svc := upload.NewService(newFakeImageRepository(), fakePresigner{}, fixedIDs{"img_123"}, fixedClock{})

	_, err := svc.Presign(context.Background(), upload.PresignInput{
		Filename:    "cat.gif",
		ContentType: "image/gif",
		Size:        2048,
	})
	if !errors.Is(err, upload.ErrInvalidUpload) {
		t.Fatalf("err = %v, want %v", err, upload.ErrInvalidUpload)
	}
}

func TestCompleteMarksImageUploaded(t *testing.T) {
	images := newFakeImageRepository()
	images.save(image.Image{ID: "img_123", Status: image.StatusPendingUpload})
	svc := upload.NewService(images, fakePresigner{}, fixedIDs{"img_123"}, fixedClock{})

	got, err := svc.Complete(context.Background(), "img_123")
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if got.Status != image.StatusUploaded {
		t.Fatalf("Status = %q, want %q", got.Status, image.StatusUploaded)
	}
}

type fakeImageRepository struct {
	images map[string]image.Image
}

func newFakeImageRepository() *fakeImageRepository {
	return &fakeImageRepository{images: map[string]image.Image{}}
}

func (r *fakeImageRepository) Save(_ context.Context, img image.Image) error {
	r.save(img)
	return nil
}

func (r *fakeImageRepository) save(img image.Image) {
	r.images[img.ID] = img
}

func (r *fakeImageRepository) FindByID(_ context.Context, id string) (image.Image, error) {
	img, ok := r.images[id]
	if !ok {
		return image.Image{}, image.ErrImageNotFound
	}
	return img, nil
}

func (r *fakeImageRepository) Update(ctx context.Context, img image.Image) error {
	return r.Save(ctx, img)
}

type fakePresigner struct{}

func (fakePresigner) PresignPut(_ context.Context, objectKey string, _ string, _ time.Duration) (string, error) {
	return "http://localhost:8080/local-upload/" + objectKey, nil
}

type fixedIDs struct {
	id string
}

func (g fixedIDs) NewID(_ string) string {
	return g.id
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
}
