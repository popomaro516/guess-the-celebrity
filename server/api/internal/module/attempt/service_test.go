package attempt_test

import (
	"context"
	"testing"
	"time"

	"github.com/tomy/guess-the-celebrity/server/api/internal/module/attempt"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/image"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/quiz"
)

func TestAnswerStoresAttemptAndHidesCorrectAnswerWhenWrong(t *testing.T) {
	repo := newFakeRepository()
	quizzes := newFakeQuizRepository()
	images := newFakeImageRepository()
	svc := attempt.NewService(repo, quizzes, images, fixedIDs{"attempt_123"}, fixedClock{})
	quizzes.save(quiz.Quiz{
		ID:       "quiz_123",
		ImageID:  "img_123",
		Answer:   "cat",
		Status:   quiz.StatusPublished,
		Choices:  []string{"cat", "dog", "fox", "rabbit"},
		Question: "これは何の動物？",
	})
	images.save(image.Image{
		ID:               "img_123",
		OriginalImageKey: "originals/anonymous/img_123/source.jpg",
		Status:           image.StatusUploaded,
	})

	got, err := svc.Answer(context.Background(), attempt.AnswerInput{QuizID: "quiz_123", Answer: "dog"})
	if err != nil {
		t.Fatalf("Answer returned error: %v", err)
	}
	if got.Correct {
		t.Fatal("Correct = true, want false")
	}
	if got.CorrectAnswer != "" {
		t.Fatalf("CorrectAnswer = %q, want hidden", got.CorrectAnswer)
	}
	if len(repo.attempts) != 1 {
		t.Fatalf("attempts = %d, want 1", len(repo.attempts))
	}
	if repo.attempts[0].IsCorrect {
		t.Fatal("stored attempt IsCorrect = true, want false")
	}
}

func TestAnswerRevealsAnswerAndOriginalImageWhenCorrect(t *testing.T) {
	repo := newFakeRepository()
	quizzes := newFakeQuizRepository()
	images := newFakeImageRepository()
	svc := attempt.NewService(repo, quizzes, images, fixedIDs{"attempt_123"}, fixedClock{})
	quizzes.save(quiz.Quiz{ID: "quiz_123", ImageID: "img_123", Answer: "cat", Status: quiz.StatusPublished})
	images.save(image.Image{ID: "img_123", OriginalImageKey: "originals/anonymous/img_123/source.jpg", Status: image.StatusUploaded})

	got, err := svc.Answer(context.Background(), attempt.AnswerInput{QuizID: "quiz_123", Answer: "cat"})
	if err != nil {
		t.Fatalf("Answer returned error: %v", err)
	}
	if !got.Correct {
		t.Fatal("Correct = false, want true")
	}
	if got.CorrectAnswer != "cat" {
		t.Fatalf("CorrectAnswer = %q, want cat", got.CorrectAnswer)
	}
	if got.OriginalImageKey != "originals/anonymous/img_123/source.jpg" {
		t.Fatalf("OriginalImageKey = %q", got.OriginalImageKey)
	}
}

type fakeRepository struct {
	attempts []attempt.Attempt
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{}
}

func (r *fakeRepository) Save(_ context.Context, a attempt.Attempt) error {
	r.attempts = append(r.attempts, a)
	return nil
}

type fakeQuizRepository struct {
	quizzes map[string]quiz.Quiz
}

func newFakeQuizRepository() *fakeQuizRepository {
	return &fakeQuizRepository{quizzes: map[string]quiz.Quiz{}}
}

func (r *fakeQuizRepository) save(q quiz.Quiz) {
	r.quizzes[q.ID] = q
}

func (r *fakeQuizRepository) FindByID(_ context.Context, id string) (quiz.Quiz, error) {
	q, ok := r.quizzes[id]
	if !ok {
		return quiz.Quiz{}, quiz.ErrQuizNotFound
	}
	return q, nil
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
