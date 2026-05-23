package aggregator

import (
	"sync"
	"time"
)

type Shard struct {
	mu     sync.Mutex
	window *Window
}

func NewShard(cfg WindowConfig) (*Shard, error) {
	window, err := NewWindow(cfg)
	if err != nil {
		return nil, err
	}

	return &Shard{
		window: window,
	}, nil
}

func (s *Shard) AddAt(event Event, now time.Time) AddResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.window.AddAt(event, now)
}

func (s *Shard) TopAt(limit int, now time.Time) []Item {
	return s.TopFilteredAt(limit, now, nil)
}

func (s *Shard) TopFilteredAt(limit int, now time.Time, include func(Item) bool) []Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.window.TopFilteredAt(limit, now, include)
}

func (s *Shard) CountAt(query string, now time.Time) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.window.CountAt(query, now)
}

func (s *Shard) UniqueQueriesAt(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.window.UniqueQueriesAt(now)
}

func (s *Shard) ActorCountersAt(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.window.ActorCountersAt(now)
}

func (s *Shard) WindowEventsAt(now time.Time) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.window.WindowEventsAt(now)
}
