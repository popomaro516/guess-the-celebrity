package quiz

import (
	"context"
	"time"

	"github.com/tomy/guess-the-celebrity/services/api/internal/module/image"
	"github.com/tomy/guess-the-celebrity/services/api/internal/module/job"
)

type Service struct {
	repo   Repository
	images ImageRepository
	queue  job.CropJobQueue
	ids    IDGenerator
	clock  Clock
}

type IDGenerator interface {
	NewID(prefix string) string
}

type Clock interface {
	Now() time.Time
}

func NewService(repo Repository, images ImageRepository, queue job.CropJobQueue, ids IDGenerator, clock Clock) *Service {
	return &Service{repo: repo, images: images, queue: queue, ids: ids, clock: clock}
}

type CreateInput struct {
	ImageID    string
	Question   string
	Answer     string
	Choices    []string
	Difficulty Difficulty
	Crop       Crop
}

type CreateOutput struct {
	ID     string
	Status Status
}

type PublicQuiz struct {
	ID              string
	Question        string
	CroppedImageKey string
	Choices         []string
	Difficulty      Difficulty
}

func (s *Service) Create(ctx context.Context, in CreateInput) (CreateOutput, error) {
	if !validCrop(in.Crop) {
		return CreateOutput{}, ErrInvalidCrop
	}
	if !validChoices(in.Answer, in.Choices) {
		return CreateOutput{}, ErrInvalidChoices
	}

	img, err := s.images.FindByID(ctx, in.ImageID)
	if err != nil {
		return CreateOutput{}, err
	}
	if img.Status != image.StatusUploaded {
		return CreateOutput{}, ErrImageNotUploaded
	}

	now := s.clock.Now()
	id := s.ids.NewID("quiz")
	croppedKey := "quizzes/" + id + "/crop.webp"
	q := Quiz{
		ID:              id,
		ImageID:         in.ImageID,
		Question:        in.Question,
		Answer:          in.Answer,
		Choices:         append([]string(nil), in.Choices...),
		Difficulty:      in.Difficulty,
		Crop:            in.Crop,
		CroppedImageKey: croppedKey,
		Status:          StatusProcessing,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.repo.Save(ctx, q); err != nil {
		return CreateOutput{}, err
	}
	if err := s.queue.EnqueueCropJob(ctx, job.CropJob{
		QuizID:         id,
		SourceImageKey: img.OriginalImageKey,
		OutputImageKey: croppedKey,
		Crop: job.Crop{
			X:      in.Crop.X,
			Y:      in.Crop.Y,
			Width:  in.Crop.Width,
			Height: in.Crop.Height,
		},
	}); err != nil {
		return CreateOutput{}, err
	}

	return CreateOutput{ID: id, Status: StatusProcessing}, nil
}

func (s *Service) Publish(ctx context.Context, quizID string) (CreateOutput, error) {
	q, err := s.repo.FindByID(ctx, quizID)
	if err != nil {
		return CreateOutput{}, err
	}
	if q.Status != StatusReady {
		return CreateOutput{}, ErrQuizNotReady
	}
	q.Status = StatusPublished
	q.UpdatedAt = s.clock.Now()
	if err := s.repo.Update(ctx, q); err != nil {
		return CreateOutput{}, err
	}
	return CreateOutput{ID: q.ID, Status: q.Status}, nil
}

func (s *Service) RandomPublished(ctx context.Context) (PublicQuiz, error) {
	q, err := s.repo.FindRandomPublished(ctx)
	if err != nil {
		return PublicQuiz{}, err
	}
	return PublicQuiz{
		ID:              q.ID,
		Question:        q.Question,
		CroppedImageKey: q.CroppedImageKey,
		Choices:         append([]string(nil), q.Choices...),
		Difficulty:      q.Difficulty,
	}, nil
}

func validCrop(c Crop) bool {
	if c.X < 0 || c.Y < 0 || c.Width <= 0 || c.Height <= 0 {
		return false
	}
	return c.X+c.Width <= 1 && c.Y+c.Height <= 1
}

func validChoices(answer string, choices []string) bool {
	if answer == "" || len(choices) != 4 {
		return false
	}
	seen := map[string]struct{}{}
	hasAnswer := false
	for _, choice := range choices {
		if choice == "" {
			return false
		}
		if _, ok := seen[choice]; ok {
			return false
		}
		seen[choice] = struct{}{}
		if choice == answer {
			hasAnswer = true
		}
	}
	return hasAnswer
}
