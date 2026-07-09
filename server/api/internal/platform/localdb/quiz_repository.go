package localdb

import (
	"context"

	"github.com/tomy/guess-the-celebrity/server/api/internal/module/quiz"
)

const quizzesCollection = "quizzes"

type QuizRepository struct {
	store *Store
}

func NewQuizRepository(store *Store) *QuizRepository {
	return &QuizRepository{store: store}
}

func (r *QuizRepository) Save(_ context.Context, q quiz.Quiz) error {
	r.store.put(quizzesCollection, q.ID, q)
	return nil
}

func (r *QuizRepository) FindByID(_ context.Context, id string) (quiz.Quiz, error) {
	doc, ok := r.store.get(quizzesCollection, id)
	if !ok {
		return quiz.Quiz{}, quiz.ErrQuizNotFound
	}
	q, ok := doc.(quiz.Quiz)
	if !ok {
		return quiz.Quiz{}, quiz.ErrQuizNotFound
	}
	return q, nil
}

func (r *QuizRepository) FindByCreatorUserID(_ context.Context, creatorUserID string) ([]quiz.Quiz, error) {
	quizzes := make([]quiz.Quiz, 0)
	for _, doc := range r.store.list(quizzesCollection) {
		q, ok := doc.(quiz.Quiz)
		if ok && q.CreatorUserID == creatorUserID {
			quizzes = append(quizzes, q)
		}
	}
	return quizzes, nil
}

func (r *QuizRepository) FindPublicQuizCandidates(_ context.Context, limit int) ([]quiz.PublicQuiz, error) {
	quizzes := make([]quiz.PublicQuiz, 0, limit)
	for _, doc := range r.store.list(quizzesCollection) {
		q, ok := doc.(quiz.Quiz)
		if ok && q.Status == quiz.StatusPublished {
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

func (r *QuizRepository) Update(ctx context.Context, q quiz.Quiz) error {
	return r.Save(ctx, q)
}

func (r *QuizRepository) Delete(_ context.Context, quizID string) error {
	r.store.delete(quizzesCollection, quizID)
	return nil
}

func (r *QuizRepository) Remove(_ context.Context, _ string) error {
	return nil
}
