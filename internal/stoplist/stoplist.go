package stoplist

import (
	"errors"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/kay-kewl/trendstream/internal/normalize"
)

var ErrEmptyTerm = errors.New("stop-list term is empty after normalization")

type Snapshot struct {
	Exact  map[string]struct{}
	Tokens map[string]struct{}
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

	return current.ContainsNormalized(term)
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
		return emptySnapshot()
	}

	return current.Clone()
}

func (s *Snapshot) ContainsNormalized(term string) bool {
	if s == nil {
		return false
	}

	if _, exists := s.Exact[term]; exists {
		return true
	}

	for _, token := range strings.Fields(term) {
		if _, exists := s.Tokens[token]; exists {
			return true
		}
	}

	return false
}

func (s *Snapshot) Clone() *Snapshot {
	if s == nil {
		return emptySnapshot()
	}

	return &Snapshot{
		Exact:  cloneSet(s.Exact),
		Tokens: cloneSet(s.Tokens),
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
	tokens := make(map[string]struct{}, len(rawTerms))

	for _, rawTerm := range rawTerms {
		term, ok := normalize.Query(rawTerm)
		if !ok {
			continue
		}

		exact[term] = struct{}{}

		// A single-token stop-list term is treated as an unwanted word and
		// suppresses any query containing that token. Multi-token terms remain
		// exact phrase rules to avoid unexpectedly hiding broad words from a
		// phrase such as "iphone 15".
		termTokens := strings.Fields(term)
		if len(termTokens) == 1 {
			tokens[termTokens[0]] = struct{}{}
		}
	}

	return &Snapshot{
		Exact:  exact,
		Tokens: tokens,
	}
}

func emptySnapshot() *Snapshot {
	return &Snapshot{
		Exact:  map[string]struct{}{},
		Tokens: map[string]struct{}{},
	}
}

func cloneSet(source map[string]struct{}) map[string]struct{} {
	if len(source) == 0 {
		return map[string]struct{}{}
	}

	copied := make(map[string]struct{}, len(source))
	for value := range source {
		copied[value] = struct{}{}
	}

	return copied
}

func containsSorted(terms []string, term string) bool {
	index, exists := slices.BinarySearch(terms, term)
	return exists && index >= 0
}
