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
	svc := quiz.NewService(repo, repo, images, &fakeObjectStore{}, queue, fixedIDs{"quiz_123"}, fixedClock{})
	images.save(image.Image{
		ID:               "img_123",
		OriginalImageKey: "originals/anonymous/img_123/source.jpg",
		Status:           image.StatusUploaded,
	})

	got, err := svc.Create(context.Background(), "user-123", quiz.CreateInput{
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
	if saved.CreatorUserID != "user-123" {
		t.Fatalf("creator user ID = %q, want user-123", saved.CreatorUserID)
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
	svc := quiz.NewService(repo, repo, newFakeImageRepository(), &fakeObjectStore{}, &fakeCropJobQueue{}, fixedIDs{"quiz_123"}, fixedClock{})

	_, err := svc.Create(context.Background(), "user-123", quiz.CreateInput{
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
	svc := quiz.NewService(repo, repo, newFakeImageRepository(), &fakeObjectStore{}, &fakeCropJobQueue{}, fixedIDs{"quiz_123"}, fixedClock{})
	repo.save(quiz.Quiz{ID: "quiz_123", CreatorUserID: "user-123", Status: quiz.StatusProcessing})

	_, err := svc.Publish(context.Background(), "user-123", "quiz_123")
	if !errors.Is(err, quiz.ErrQuizNotReady) {
		t.Fatalf("err = %v, want %v", err, quiz.ErrQuizNotReady)
	}
}

func TestPublishRejectsUserWhoIsNotCreator(t *testing.T) {
	repo := newFakeRepository()
	svc := quiz.NewService(repo, repo, newFakeImageRepository(), &fakeObjectStore{}, &fakeCropJobQueue{}, fixedIDs{"quiz_123"}, fixedClock{})
	repo.save(quiz.Quiz{ID: "quiz_123", CreatorUserID: "owner-123", Status: quiz.StatusReady})

	_, err := svc.Publish(context.Background(), "other-user", "quiz_123")
	if !errors.Is(err, quiz.ErrPublishForbidden) {
		t.Fatalf("err = %v, want %v", err, quiz.ErrPublishForbidden)
	}

	saved, err := repo.FindByID(context.Background(), "quiz_123")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if saved.Status != quiz.StatusReady {
		t.Fatalf("status = %s, want %s", saved.Status, quiz.StatusReady)
	}
}

func TestPublishAllowsCreator(t *testing.T) {
	repo := newFakeRepository()
	svc := quiz.NewService(repo, repo, newFakeImageRepository(), &fakeObjectStore{}, &fakeCropJobQueue{}, fixedIDs{"quiz_123"}, fixedClock{})
	repo.save(quiz.Quiz{ID: "quiz_123", CreatorUserID: "user-123", Status: quiz.StatusReady})

	got, err := svc.Publish(context.Background(), "user-123", "quiz_123")
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if got.Status != quiz.StatusPublished {
		t.Fatalf("status = %s, want %s", got.Status, quiz.StatusPublished)
	}
}

func TestListOwnedReturnsOnlyCreatorsQuizzesNewestFirst(t *testing.T) {
	repo := newFakeRepository()
	svc := quiz.NewService(repo, repo, newFakeImageRepository(), &fakeObjectStore{}, &fakeCropJobQueue{}, fixedIDs{"quiz_123"}, fixedClock{})
	repo.save(quiz.Quiz{
		ID:            "quiz_old",
		CreatorUserID: "user-123",
		Question:      "古い問題",
		Status:        quiz.StatusReady,
		CreatedAt:     time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	})
	repo.save(quiz.Quiz{
		ID:            "quiz_other",
		CreatorUserID: "other-user",
		Question:      "別ユーザーの問題",
		Status:        quiz.StatusPublished,
		CreatedAt:     time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
	})
	repo.save(quiz.Quiz{
		ID:            "quiz_new",
		CreatorUserID: "user-123",
		Question:      "新しい問題",
		Status:        quiz.StatusProcessing,
		CreatedAt:     time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	})

	got, err := svc.ListOwned(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("ListOwned returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(ListOwned) = %d, want 2", len(got))
	}
	if got[0].ID != "quiz_new" || got[1].ID != "quiz_old" {
		t.Fatalf("quiz IDs = [%s, %s], want [quiz_new, quiz_old]", got[0].ID, got[1].ID)
	}
}

func TestRandomPublishedReturnsFeedProjectionWithoutLoadingQuiz(t *testing.T) {
	repo := newFakeRepository()
	feed := &fakePublicFeedRepository{
		quizzes: []quiz.PublicQuiz{
			{
				ID:              "quiz_feed",
				Question:        "feedの問題",
				CroppedImageKey: "quizzes/quiz_feed/crop.webp",
				Choices:         []string{"A", "B", "C", "D"},
				Difficulty:      quiz.DifficultyHard,
			},
		},
	}
	svc := quiz.NewService(repo, feed, newFakeImageRepository(), &fakeObjectStore{}, &fakeCropJobQueue{}, fixedIDs{"quiz_123"}, fixedClock{})

	got, err := svc.RandomPublished(context.Background())
	if err != nil {
		t.Fatalf("RandomPublished returned error: %v", err)
	}
	if got.ID != "quiz_feed" || got.Question != "feedの問題" {
		t.Fatalf("unexpected public quiz: %+v", got)
	}
	if repo.findByIDCalls != 0 {
		t.Fatalf("FindByID calls = %d, want 0", repo.findByIDCalls)
	}
}

func TestDeleteRemovesOwnedQuizFeedEntryAndObjects(t *testing.T) {
	repo := newFakeRepository()
	feed := &fakePublicFeedRepository{
		quizzes: []quiz.PublicQuiz{{ID: "quiz_123"}, {ID: "quiz_other"}},
	}
	images := newFakeImageRepository()
	objects := &fakeObjectStore{}
	svc := quiz.NewService(repo, feed, images, objects, &fakeCropJobQueue{}, fixedIDs{"quiz_123"}, fixedClock{})
	repo.save(quiz.Quiz{
		ID:              "quiz_123",
		CreatorUserID:   "user-123",
		ImageID:         "img_123",
		CroppedImageKey: "quizzes/quiz_123/crop.webp",
		Status:          quiz.StatusPublished,
	})
	images.save(image.Image{
		ID:               "img_123",
		OriginalImageKey: "originals/anonymous/img_123/source.jpg",
		Status:           image.StatusUploaded,
	})

	err := svc.Delete(context.Background(), "user-123", "quiz_123")
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := repo.FindByID(context.Background(), "quiz_123"); !errors.Is(err, quiz.ErrQuizNotFound) {
		t.Fatalf("FindByID err = %v, want %v", err, quiz.ErrQuizNotFound)
	}
	if len(feed.quizzes) != 1 || feed.quizzes[0].ID != "quiz_other" {
		t.Fatalf("feed quizzes = %+v, want only quiz_other", feed.quizzes)
	}
	wantDeleted := []string{"quizzes/quiz_123/crop.webp", "originals/anonymous/img_123/source.jpg"}
	if !equalStrings(objects.deleted, wantDeleted) {
		t.Fatalf("deleted objects = %+v, want %+v", objects.deleted, wantDeleted)
	}
}

func TestDeleteRejectsUserWhoIsNotCreator(t *testing.T) {
	repo := newFakeRepository()
	images := newFakeImageRepository()
	objects := &fakeObjectStore{}
	svc := quiz.NewService(repo, repo, images, objects, &fakeCropJobQueue{}, fixedIDs{"quiz_123"}, fixedClock{})
	repo.save(quiz.Quiz{
		ID:              "quiz_123",
		CreatorUserID:   "owner-123",
		ImageID:         "img_123",
		CroppedImageKey: "quizzes/quiz_123/crop.webp",
	})

	err := svc.Delete(context.Background(), "other-user", "quiz_123")
	if !errors.Is(err, quiz.ErrDeleteForbidden) {
		t.Fatalf("err = %v, want %v", err, quiz.ErrDeleteForbidden)
	}
	if _, err := repo.FindByID(context.Background(), "quiz_123"); err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if len(objects.deleted) != 0 {
		t.Fatalf("deleted objects = %+v, want none", objects.deleted)
	}
}

func TestDeleteContinuesWhenImageMetadataIsMissing(t *testing.T) {
	repo := newFakeRepository()
	objects := &fakeObjectStore{}
	svc := quiz.NewService(repo, repo, newFakeImageRepository(), objects, &fakeCropJobQueue{}, fixedIDs{"quiz_123"}, fixedClock{})
	repo.save(quiz.Quiz{
		ID:              "quiz_123",
		CreatorUserID:   "user-123",
		ImageID:         "img_missing",
		CroppedImageKey: "quizzes/quiz_123/crop.webp",
	})

	err := svc.Delete(context.Background(), "user-123", "quiz_123")
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := repo.FindByID(context.Background(), "quiz_123"); !errors.Is(err, quiz.ErrQuizNotFound) {
		t.Fatalf("FindByID err = %v, want %v", err, quiz.ErrQuizNotFound)
	}
	wantDeleted := []string{"quizzes/quiz_123/crop.webp"}
	if !equalStrings(objects.deleted, wantDeleted) {
		t.Fatalf("deleted objects = %+v, want %+v", objects.deleted, wantDeleted)
	}
}

type fakeRepository struct {
	quizzes       map[string]quiz.Quiz
	findByIDCalls int
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
	r.findByIDCalls++
	q, ok := r.quizzes[id]
	if !ok {
		return quiz.Quiz{}, quiz.ErrQuizNotFound
	}
	return q, nil
}

func (r *fakeRepository) FindByCreatorUserID(_ context.Context, creatorUserID string) ([]quiz.Quiz, error) {
	quizzes := make([]quiz.Quiz, 0)
	for _, q := range r.quizzes {
		if q.CreatorUserID == creatorUserID {
			quizzes = append(quizzes, q)
		}
	}
	return quizzes, nil
}

func (r *fakeRepository) FindPublicQuizCandidates(_ context.Context, limit int) ([]quiz.PublicQuiz, error) {
	quizzes := make([]quiz.PublicQuiz, 0, limit)
	for _, q := range r.quizzes {
		if q.Status == quiz.StatusPublished {
			quizzes = append(quizzes, quiz.PublicQuiz{
				ID:              q.ID,
				Question:        q.Question,
				CroppedImageKey: q.CroppedImageKey,
				Choices:         append([]string(nil), q.Choices...),
				Difficulty:      q.Difficulty,
			})
			if len(quizzes) == limit {
				return quizzes, nil
			}
		}
	}
	return quizzes, nil
}

func (r *fakeRepository) Update(_ context.Context, q quiz.Quiz) error {
	r.quizzes[q.ID] = q
	return nil
}

func (r *fakeRepository) Delete(_ context.Context, quizID string) error {
	delete(r.quizzes, quizID)
	return nil
}

func (r *fakeRepository) Remove(_ context.Context, _ string) error {
	return nil
}

type fakePublicFeedRepository struct {
	quizzes []quiz.PublicQuiz
}

func (r *fakePublicFeedRepository) FindPublicQuizCandidates(_ context.Context, limit int) ([]quiz.PublicQuiz, error) {
	if len(r.quizzes) < limit {
		limit = len(r.quizzes)
	}
	return append([]quiz.PublicQuiz(nil), r.quizzes[:limit]...), nil
}

func (r *fakePublicFeedRepository) Remove(_ context.Context, quizID string) error {
	filtered := make([]quiz.PublicQuiz, 0, len(r.quizzes))
	for _, q := range r.quizzes {
		if q.ID != quizID {
			filtered = append(filtered, q)
		}
	}
	r.quizzes = filtered
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

type fakeObjectStore struct {
	deleted []string
}

func (s *fakeObjectStore) Delete(_ context.Context, objectKey string) error {
	s.deleted = append(s.deleted, objectKey)
	return nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
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
