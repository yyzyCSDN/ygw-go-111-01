package record

import (
	"sync"

	"platformscreendoor/internal/event"
)

type Store struct {
	mu      sync.Mutex
	records []Record
	seen    map[uint64]bool
	limit   int
}

func NewStore(limit int) *Store {
	return &Store{
		seen:  make(map[uint64]bool),
		limit: limit,
	}
}

func (s *Store) Append(e event.Event) Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := FromEvent(e)
	if s.seen[r.Fingerprint] {
		return r
	}
	s.seen[r.Fingerprint] = true
	s.records = append(s.records, r)
	if s.limit > 0 && len(s.records) > s.limit {
		s.records = s.records[len(s.records)-s.limit:]
	}
	return r
}

func (s *Store) Query(after uint64, limit int) []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Record, 0)
	for _, r := range s.records {
		if r.Seq > after {
			out = append(out, r)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out
}

func (s *Store) ByDoor(doorID string, after uint64, limit int) []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Record, 0)
	for _, r := range s.records {
		if r.DoorID == doorID && r.Seq > after {
			out = append(out, r)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out
}

func (s *Store) All() []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Record, len(s.records))
	copy(out, s.records)
	return out
}

func (s *Store) Size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.records)
}

func (s *Store) Summary() map[event.Type]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[event.Type]int)
	for _, r := range s.records {
		out[r.Type]++
	}
	return out
}
