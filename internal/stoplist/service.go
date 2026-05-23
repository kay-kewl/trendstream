package stoplist

import (
	"sync"
)

type Service struct {
	mu    sync.Mutex
	list  *StopList
	store Store
}

func NewService(store Store) (*Service, error) {
	terms, err := store.Load()
	if err != nil {
		return nil, err
	}

	return &Service{
		list:  New(terms),
		store: store,
	}, nil
}

func (s *Service) Contains(rawTerm string) bool {
	return s.list.Contains(rawTerm)
}

func (s *Service) Terms() []string {
	return s.list.Terms()
}

func (s *Service) Add(rawTerm string) (string, bool, error) {
	term, err := NormalizeTerm(rawTerm)
	if err != nil {
		return "", false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.list.Terms()
	if containsSorted(current, term) {
		return term, false, nil
	}

	next := append(current, term)

	if err := s.store.Save(next); err != nil {
		return "", false, err
	}

	s.list.Replace(next)

	return term, true, nil
}

func (s *Service) Remove(rawTerm string) (string, bool, error) {
	term, err := NormalizeTerm(rawTerm)
	if err != nil {
		return "", false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.list.Terms()
	if !containsSorted(current, term) {
		return term, false, nil
	}

	next := make([]string, 0, len(current)-1)
	for _, existingTerm := range current {
		if existingTerm == term {
			continue
		}

		next = append(next, existingTerm)
	}

	if err := s.store.Save(next); err != nil {
		return "", false, err
	}

	s.list.Replace(next)

	return term, true, nil
}
