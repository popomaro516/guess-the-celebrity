package image

import (
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
