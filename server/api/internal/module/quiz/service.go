package quiz

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"math/big"
	"sort"
	"time"

	"github.com/tomy/guess-the-celebrity/server/api/internal/module/image"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/job"
)

type Service struct {
	repo       Repository
	publicFeed PublicFeedRepository
	images     ImageRepository
	objects    ObjectStore
	queue      job.CropJobQueue
	ids        IDGenerator
	clock      Clock
}

type IDGenerator interface {
	NewID(prefix string) string
}

type Clock interface {
	Now() time.Time
}

func NewService(repo Repository, publicFeed PublicFeedRepository, images ImageRepository, objects ObjectStore, queue job.CropJobQueue, ids IDGenerator, clock Clock) *Service {
	return &Service{repo: repo, publicFeed: publicFeed, images: images, objects: objects, queue: queue, ids: ids, clock: clock}
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

func (s *Service) Create(ctx context.Context, creatorUserID string, in CreateInput) (CreateOutput, error) {
	if creatorUserID == "" {
		return CreateOutput{}, errors.New("creator user ID is required")
	}
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
		CreatorUserID:   creatorUserID,
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

func (s *Service) Publish(ctx context.Context, creatorUserID, quizID string) (CreateOutput, error) {
	q, err := s.repo.FindByID(ctx, quizID)
	if err != nil {
		return CreateOutput{}, err
	}
	if q.CreatorUserID != creatorUserID {
		return CreateOutput{}, ErrPublishForbidden
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

func (s *Service) ListOwned(ctx context.Context, creatorUserID string) ([]Quiz, error) {
	if creatorUserID == "" {
		return nil, errors.New("creator user ID is required")
	}
	quizzes, err := s.repo.FindByCreatorUserID(ctx, creatorUserID)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(quizzes, func(i, j int) bool {
		return quizzes[i].CreatedAt.After(quizzes[j].CreatedAt)
	})
	return quizzes, nil
}

func (s *Service) Delete(ctx context.Context, creatorUserID, quizID string) error {
	if creatorUserID == "" {
		return errors.New("creator user ID is required")
	}
	q, err := s.repo.FindByID(ctx, quizID)
	if err != nil {
		return err
	}
	if q.CreatorUserID != creatorUserID {
		return ErrDeleteForbidden
	}

	originalImageKey := ""
	img, err := s.images.FindByID(ctx, q.ImageID)
	if err != nil && !errors.Is(err, image.ErrImageNotFound) {
		return err
	}
	if err == nil {
		originalImageKey = img.OriginalImageKey
	}
	if err := s.repo.Delete(ctx, q.ID); err != nil {
		return err
	}
	if err := s.publicFeed.Remove(ctx, q.ID); err != nil {
		return err
	}
	if q.CroppedImageKey != "" {
		if err := s.objects.Delete(ctx, q.CroppedImageKey); err != nil {
			return err
		}
	}
	if originalImageKey != "" {
		if err := s.objects.Delete(ctx, originalImageKey); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) RandomPublished(ctx context.Context, count int) ([]PublicQuiz, error) {
	quizzes, err := s.publicFeed.FindPublicQuizCandidates(ctx, 10)
	if err != nil {
		return nil, err
	}
	if len(quizzes) == 0 {
		return nil, ErrQuizNotFound
	}
	if count > len(quizzes) {
		count = len(quizzes)
	}

	selected := append([]PublicQuiz(nil), quizzes...)
	for index := 0; index < count; index++ {
		offset, err := randomIndex(len(selected) - index)
		if err != nil {
			return nil, err
		}
		swapIndex := index + offset
		selected[index], selected[swapIndex] = selected[swapIndex], selected[index]
		selected[index].Choices = append([]string(nil), selected[index].Choices...)
	}
	return selected[:count], nil
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

func randomIndex(length int) (int, error) {
	n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(length)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}
