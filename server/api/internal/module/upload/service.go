package upload

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/tomy/guess-the-celebrity/server/api/internal/module/image"
)

const (
	maxImageSize      = 10 * 1024 * 1024
	presignExpiration = 5 * time.Minute
)

type Service struct {
	images    ImageRepository
	presigner Presigner
	objects   ObjectStore
	ids       IDGenerator
	clock     Clock
}

type IDGenerator interface {
	NewID(prefix string) string
}

type Clock interface {
	Now() time.Time
}

func NewService(images ImageRepository, presigner Presigner, objects ObjectStore, ids IDGenerator, clock Clock) *Service {
	return &Service{images: images, presigner: presigner, objects: objects, ids: ids, clock: clock}
}

type PresignInput struct {
	Filename    string
	ContentType string
	Size        int64
}

type PresignOutput struct {
	ImageID   string
	UploadURL string
	ObjectKey string
	ExpiresIn int
}

func (s *Service) Presign(ctx context.Context, in PresignInput) (PresignOutput, error) {
	if !allowedContentType(in.ContentType) || in.Size <= 0 || in.Size > maxImageSize {
		return PresignOutput{}, ErrInvalidUpload
	}

	id := s.ids.NewID("img")
	objectKey := "originals/anonymous/" + id + "/source" + extension(in.Filename, in.ContentType)
	url, err := s.presigner.PresignPut(ctx, objectKey, in.ContentType, presignExpiration)
	if err != nil {
		return PresignOutput{}, err
	}

	now := s.clock.Now()
	if err := s.images.Save(ctx, image.Image{
		ID:               id,
		OwnerUserID:      "anonymous",
		OriginalImageKey: objectKey,
		ContentType:      in.ContentType,
		Size:             in.Size,
		Status:           image.StatusPendingUpload,
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		return PresignOutput{}, err
	}

	return PresignOutput{ImageID: id, UploadURL: url, ObjectKey: objectKey, ExpiresIn: int(presignExpiration.Seconds())}, nil
}

func (s *Service) Complete(ctx context.Context, imageID string) (image.Image, error) {
	img, err := s.images.FindByID(ctx, imageID)
	if err != nil {
		return image.Image{}, err
	}
	exists, err := s.objects.Exists(ctx, img.OriginalImageKey)
	if err != nil {
		return image.Image{}, err
	}
	if !exists {
		return image.Image{}, ErrUploadObjectNotFound
	}
	img.Status = image.StatusUploaded
	img.UpdatedAt = s.clock.Now()
	if err := s.images.Update(ctx, img); err != nil {
		return image.Image{}, err
	}
	return img, nil
}

func allowedContentType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func extension(filename, contentType string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return ext
	}
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}
