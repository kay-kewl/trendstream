package stoplist

import (
	"errors"
	"slices"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/kay-kewl/trendstream/internal/normalize"
)

var ErrEmptyTerm = errors.New("stop-list term is empty after normalization")

type Snapshot struct {
	Exact map[string]struct{}
}

type StopList struct {
	mu      sync.Mutex
	current atomic.Pointer[Snapshot]
}

func New(terms []string) *StopList {
	stopList := &StopList{}
	stopList.Replace(terms)

	return stopList
}

func (s *StopList) Contains(rawTerm string) bool {
	term, ok := normalize.Query(rawTerm)
	if !ok {
		return false
	}

	current := s.current.Load()
	if current == nil {
		return false
	}

	_, exists := current.Exact[term]
	return exists
}

func (s *StopList) Terms() []string {
	current := s.current.Load()
	if current == nil || len(current.Exact) == 0 {
		return []string{}
	}

	terms := make([]string, 0, len(current.Exact))
	for term := range current.Exact {
		terms = append(terms, term)
	}

	sort.Strings(terms)

	return terms
}

func (s *StopList) Replace(rawTerms []string) {
	next := buildSnapshot(rawTerms)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.current.Store(next)
}

func (s *StopList) Snapshot() *Snapshot {
	current := s.current.Load()
	if current == nil {
		return &Snapshot{
			Exact: map[string]struct{}{},
		}
	}

	copied := make(map[string]struct{}, len(current.Exact))
	for term := range current.Exact {
		copied[term] = struct{}{}
	}

	return &Snapshot{
		Exact: copied,
	}
}

func NormalizeTerm(rawTerm string) (string, error) {
	term, ok := normalize.Query(rawTerm)
	if !ok {
		return "", ErrEmptyTerm
	}

	return term, nil
}

func buildSnapshot(rawTerms []string) *Snapshot {
	exact := make(map[string]struct{}, len(rawTerms))

	for _, rawTerm := range rawTerms {
		term, ok := normalize.Query(rawTerm)
		if !ok {
			continue
		}

		exact[term] = struct{}{}
	}

	return &Snapshot{
		Exact: exact,
	}
}

func containsSorted(terms []string, term string) bool {
	index, exists := slices.BinarySearch(terms, term)
	return exists && index >= 0
}
