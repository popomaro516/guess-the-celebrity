package quiz

import (
	"context"
	"errors"
	"time"

	"github.com/tomy/guess-the-celebrity/services/api/internal/module/image"
)

var (
	ErrInvalidCrop      = errors.New("invalid crop")
	ErrInvalidChoices   = errors.New("invalid choices")
	ErrImageNotUploaded = errors.New("image is not uploaded")
	ErrQuizNotFound     = errors.New("quiz not found")
	ErrQuizNotReady     = errors.New("quiz is not ready")
)

type Status string

const (
	StatusProcessing Status = "processing"
	StatusReady      Status = "ready"
	StatusPublished  Status = "published"
	StatusFailed     Status = "failed"
	StatusArchived   Status = "archived"
)

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyNormal Difficulty = "normal"
	DifficultyHard   Difficulty = "hard"
)

type Crop struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

type Quiz struct {
	ID              string
	CreatorUserID   string
	ImageID         string
	Question        string
	Answer          string
	Choices         []string
	Difficulty      Difficulty
	Crop            Crop
	CroppedImageKey string
	Status          Status
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Repository interface {
	Save(ctx context.Context, q Quiz) error
	FindByID(ctx context.Context, id string) (Quiz, error)
	FindRandomPublished(ctx context.Context) (Quiz, error)
	Update(ctx context.Context, q Quiz) error
}

type ImageRepository interface {
	FindByID(ctx context.Context, id string) (image.Image, error)
}

type IDGenerator interface {
	NewID(prefix string) string
}

type Clock interface {
	Now() time.Time
}
