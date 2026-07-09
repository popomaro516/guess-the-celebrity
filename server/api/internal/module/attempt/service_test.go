package attempt_test

import (
	"context"
	"testing"

	"github.com/tomy/guess-the-celebrity/server/api/internal/module/attempt"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/image"
	"github.com/tomy/guess-the-celebrity/server/api/internal/module/quiz"
)

func TestAnswerRevealsAnswerAndOriginalImageWhenWrong(t *testing.T) {
	quizzes := newFakeQuizRepository()
	images := newFakeImageRepository()
	svc := attempt.NewService(quizzes, images)
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
	if got.CorrectAnswer != "cat" {
		t.Fatalf("CorrectAnswer = %q, want cat", got.CorrectAnswer)
	}
	if got.OriginalImageKey != "originals/anonymous/img_123/source.jpg" {
		t.Fatalf("OriginalImageKey = %q", got.OriginalImageKey)
	}
}

func TestAnswerRevealsAnswerAndOriginalImageWhenCorrect(t *testing.T) {
	quizzes := newFakeQuizRepository()
	images := newFakeImageRepository()
	svc := attempt.NewService(quizzes, images)
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
