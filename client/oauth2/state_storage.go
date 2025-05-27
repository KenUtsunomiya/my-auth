package oauth2

import (
	"fmt"
	"sync"
)

type StateStorage interface {
	Load(state string) string
	Save(state string, originalUrl string)
}

type InMemoryStateStorage struct {
	mu      sync.RWMutex
	storage map[string]string
}

func NewInMemoryStateStorage() *InMemoryStateStorage {
	return &InMemoryStateStorage{storage: make(map[string]string)}
}

func (s *InMemoryStateStorage) Load(state string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.storage[state]
	return val, ok
}

func (s *InMemoryStateStorage) Save(state string, originalUrl string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.storage[state]; exists {
		return fmt.Errorf("state %s already exists")
	}
	s.storage[state] = originalUrl
	return nil
}
