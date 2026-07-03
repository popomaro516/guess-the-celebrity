package image

import (
	"context"
	"errors"
	"time"
)

var ErrImageNotFound = errors.New("image not found")

type Status string

const (
	StatusPendingUpload Status = "pending_upload"
	StatusUploaded      Status = "uploaded"
)

type Image struct {
	ID               string
	OwnerUserID      string
	OriginalImageKey string
	ContentType      string
	Size             int64
	Status           Status
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Repository interface {
	Save(ctx context.Context, img Image) error
	FindByID(ctx context.Context, id string) (Image, error)
	Update(ctx context.Context, img Image) error
}
