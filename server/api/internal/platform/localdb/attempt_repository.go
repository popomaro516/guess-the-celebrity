package localdb

import (
	"context"

	"github.com/tomy/guess-the-celebrity/server/api/internal/module/attempt"
)

const attemptsCollection = "attempts"

type AttemptRepository struct {
	store *Store
}

func NewAttemptRepository(store *Store) *AttemptRepository {
	return &AttemptRepository{store: store}
}

func (r *AttemptRepository) Save(_ context.Context, a attempt.Attempt) error {
	r.store.put(attemptsCollection, a.ID, a)
	return nil
}
