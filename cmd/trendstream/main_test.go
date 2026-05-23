package main

import (
	"testing"

	"github.com/kay-kewl/trendstream/internal/aggregator"
	"github.com/kay-kewl/trendstream/internal/stoplist"
)

type memoryStopListStore struct {
	terms []string
}

func (s *memoryStopListStore) Load() ([]string, error) {
	return append([]string(nil), s.terms...), nil
}

func (s *memoryStopListStore) Save(terms []string) error {
	s.terms = append([]string(nil), terms...)
	return nil
}

func TestFilterStopListedItemsRemovesExactAndTokenMatches(t *testing.T) {
	t.Parallel()

	service, err := stoplist.NewService(&memoryStopListStore{
		terms: []string{"casino", "manual iphone 15"},
	})
	if err != nil {
		t.Fatalf("failed to create stop-list service: %v", err)
	}

	items := []aggregator.Item{
		{Query: "laptop", Count: 10},
		{Query: "best casino online", Count: 9},
		{Query: "manual iphone 15", Count: 8},
		{Query: "iphone case", Count: 7},
	}

	filtered := filterStopListedItems(items, service)
	want := []aggregator.Item{
		{Query: "laptop", Count: 10},
		{Query: "iphone case", Count: 7},
	}

	if len(filtered) != len(want) {
		t.Fatalf("filtered length mismatch: got %d, want %d; filtered=%#v", len(filtered), len(want), filtered)
	}

	for i := range want {
		if filtered[i] != want[i] {
			t.Fatalf("filtered item %d mismatch: got %#v, want %#v", i, filtered[i], want[i])
		}
	}
}
