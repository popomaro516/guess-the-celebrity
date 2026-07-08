package localdb

import "sync"

type Store struct {
	mu          sync.RWMutex
	collections map[string]map[string]any
}

func NewStore() *Store {
	return &Store{collections: map[string]map[string]any{}}
}

func (s *Store) put(collection, id string, doc any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.collections[collection]; !ok {
		s.collections[collection] = map[string]any{}
	}
	s.collections[collection][id] = doc
}

func (s *Store) get(collection, id string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, ok := s.collections[collection]
	if !ok {
		return nil, false
	}
	doc, ok := items[id]
	return doc, ok
}

func (s *Store) list(collection string) []any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.collections[collection]
	out := make([]any, 0, len(items))
	for _, doc := range items {
		out = append(out, doc)
	}
	return out
}
