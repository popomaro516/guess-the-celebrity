package quiz_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tomy/guess-the-celebrity/server/api/internal/module/image"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/job"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/quiz"
)

func TestCreateStoresProcessingQuizAndEnqueuesCropJob(t *testing.T) {
	repo := newFakeRepository()
	images := newFakeImageRepository()
	queue := &fakeCropJobQueue{}
	svc := quiz.NewService(repo, repo, images, queue, fixedIDs{"quiz_123"}, fixedClock{})
	images.save(image.Image{
		ID:               "img_123",
		OriginalImageKey: "originals/anonymous/img_123/source.jpg",
		Status:           image.StatusUploaded,
	})

	got, err := svc.Create(context.Background(), quiz.CreateInput{
		ImageID:    "img_123",
		Question:   "これは何の動物？",
		Answer:     "cat",
		Choices:    []string{"cat", "dog", "fox", "rabbit"},
		Difficulty: quiz.DifficultyNormal,
		Crop:       quiz.Crop{X: 0.24, Y: 0.18, Width: 0.32, Height: 0.28},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if got.ID != "quiz_123" || got.Status != quiz.StatusProcessing {
		t.Fatalf("unexpected quiz summary: %+v", got)
	}

	saved, err := repo.FindByID(context.Background(), "quiz_123")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if saved.Status != quiz.StatusProcessing {
		t.Fatalf("status = %s, want %s", saved.Status, quiz.StatusProcessing)
	}
	if len(queue.jobs) != 1 {
		t.Fatalf("queued jobs = %d, want 1", len(queue.jobs))
	}
	wantJob := job.CropJob{
		QuizID:         "quiz_123",
		SourceImageKey: "originals/anonymous/img_123/source.jpg",
		OutputImageKey: "quizzes/quiz_123/crop.webp",
		Crop:           job.Crop{X: 0.24, Y: 0.18, Width: 0.32, Height: 0.28},
	}
	if queue.jobs[0] != wantJob {
		t.Fatalf("job = %+v, want %+v", queue.jobs[0], wantJob)
	}
}

func TestCreateRejectsInvalidCrop(t *testing.T) {
	repo := newFakeRepository()
	svc := quiz.NewService(repo, repo, newFakeImageRepository(), &fakeCropJobQueue{}, fixedIDs{"quiz_123"}, fixedClock{})

	_, err := svc.Create(context.Background(), quiz.CreateInput{
		ImageID:    "img_123",
		Question:   "これは何の動物？",
		Answer:     "cat",
		Choices:    []string{"cat", "dog", "fox", "rabbit"},
		Difficulty: quiz.DifficultyNormal,
		Crop:       quiz.Crop{X: 0.8, Y: 0.2, Width: 0.3, Height: 0.3},
	})
	if !errors.Is(err, quiz.ErrInvalidCrop) {
		t.Fatalf("err = %v, want %v", err, quiz.ErrInvalidCrop)
	}
}

func TestPublishRequiresReadyQuiz(t *testing.T) {
	repo := newFakeRepository()
	svc := quiz.NewService(repo, repo, newFakeImageRepository(), &fakeCropJobQueue{}, fixedIDs{"quiz_123"}, fixedClock{})
	repo.save(quiz.Quiz{ID: "quiz_123", Status: quiz.StatusProcessing})

	_, err := svc.Publish(context.Background(), "quiz_123")
	if !errors.Is(err, quiz.ErrQuizNotReady) {
		t.Fatalf("err = %v, want %v", err, quiz.ErrQuizNotReady)
	}
}

type fakeRepository struct {
	quizzes map[string]quiz.Quiz
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{quizzes: map[string]quiz.Quiz{}}
}

func (r *fakeRepository) Save(_ context.Context, q quiz.Quiz) error {
	r.save(q)
	return nil
}

func (r *fakeRepository) save(q quiz.Quiz) {
	r.quizzes[q.ID] = q
}

func (r *fakeRepository) FindByID(_ context.Context, id string) (quiz.Quiz, error) {
	q, ok := r.quizzes[id]
	if !ok {
		return quiz.Quiz{}, quiz.ErrQuizNotFound
	}
	return q, nil
}

func (r *fakeRepository) FindPublicQuizCandidateIDs(_ context.Context, limit int) ([]string, error) {
	ids := make([]string, 0, limit)
	for _, q := range r.quizzes {
		if q.Status == quiz.StatusPublished {
			ids = append(ids, q.ID)
			if len(ids) == limit {
				return ids, nil
			}
		}
	}
	return ids, nil
}

func (r *fakeRepository) Update(_ context.Context, q quiz.Quiz) error {
	r.quizzes[q.ID] = q
	return nil
}

type fakeImageRepository struct {
	images map[string]image.Image
}

func newFakeImageRepository() *fakeImageRepository {
	return &fakeImageRepository{images: map[string]image.Image{}}
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

type fakeCropJobQueue struct {
	jobs []job.CropJob
}

func (q *fakeCropJobQueue) EnqueueCropJob(_ context.Context, cropJob job.CropJob) error {
	q.jobs = append(q.jobs, cropJob)
	return nil
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
