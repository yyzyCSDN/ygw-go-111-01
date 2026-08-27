package door

import (
	"fmt"
	"sync"
)

type Store struct {
	mu    sync.RWMutex
	doors map[string]*Door
}

func NewStore() *Store {
	return &Store{doors: make(map[string]*Door)}
}

func (s *Store) Add(d *Door) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doors[d.ID] = d
}

func (s *Store) AddAll(doors ...*Door) {
	for _, d := range doors {
		s.Add(d)
	}
}

func (s *Store) Get(id string) (*Door, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.doors[id]
	return d, ok
}

func (s *Store) All() []*Door {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Door, 0, len(s.doors))
	for _, d := range s.doors {
		out = append(out, d)
	}
	return out
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.doors)
}

func NewDoors(count int, cfg Config) []*Door {
	cfg = NormalizeConfig(cfg)
	doors := make([]*Door, 0, count)
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("PSD%02d", i+1)
		doors = append(doors, New(id, 1, i, cfg))
	}
	return doors
}
