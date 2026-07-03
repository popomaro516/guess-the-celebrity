package localdb

import (
	"context"

	"github.com/tomy/guess-the-celebrity/services/api/internal/module/quiz"
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

func (r *QuizRepository) FindRandomPublished(_ context.Context) (quiz.Quiz, error) {
	for _, doc := range r.store.list(quizzesCollection) {
		q, ok := doc.(quiz.Quiz)
		if ok && q.Status == quiz.StatusPublished {
			return q, nil
		}
	}
	return quiz.Quiz{}, quiz.ErrQuizNotFound
}

func (r *QuizRepository) Update(ctx context.Context, q quiz.Quiz) error {
	return r.Save(ctx, q)
}
