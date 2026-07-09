package localstorage

import "context"

type ObjectStore struct{}

func NewObjectStore() ObjectStore {
	return ObjectStore{}
}

func (ObjectStore) Exists(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func (ObjectStore) Delete(_ context.Context, _ string) error {
	return nil
}
