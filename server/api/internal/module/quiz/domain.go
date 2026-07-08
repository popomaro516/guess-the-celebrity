package quiz

import (
	"errors"
	"time"
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
