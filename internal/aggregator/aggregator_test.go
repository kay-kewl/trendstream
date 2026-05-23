package aggregator

import (
	"errors"
	"testing"
	"time"
)

func TestNewUsesDefaultShardCount(t *testing.T) {
	t.Parallel()

	aggregator := newTestAggregator(t, Config{})

	if aggregator.ShardCount() != DefaultShardCount {
		t.Fatalf("shard count mismatch: got %d, want %d", aggregator.ShardCount(), DefaultShardCount)
	}
}

func TestNewRejectsNegativeShardCount(t *testing.T) {
	t.Parallel()

	_, err := New(Config{
		ShardCount: -1,
		Window:     DefaultWindowConfig(),
	})

	if !errors.Is(err, ErrInvalidShardCount) {
		t.Fatalf("expected ErrInvalidShardCount, got %v", err)
	}
}

func TestAggregatorRoutesSameQueryToSameShard(t *testing.T) {
	t.Parallel()

	aggregator := newTestAggregator(t, Config{
		ShardCount: 8,
		Window:     DefaultWindowConfig(),
	})

	first := aggregator.ShardIndex("iphone 15")
	second := aggregator.ShardIndex("iphone 15")

	if first != second {
		t.Fatalf("same query should be routed to same shard: first=%d second=%d", first, second)
	}
}

func TestAggregatorShardIndexIsInsideRange(t *testing.T) {
	t.Parallel()

	aggregator := newTestAggregator(t, Config{
		ShardCount: 8,
		Window:     DefaultWindowConfig(),
	})

	for _, query := range []string{
		"",
		"iphone 15",
		"кроссовки женские",
		"query with several words",
	} {
		index := aggregator.ShardIndex(query)

		if index < 0 || index >= aggregator.ShardCount() {
			t.Fatalf("shard index out of range for %q: got %d", query, index)
		}
	}
}

func TestAggregatorAddAndTop(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	aggregator := newTestAggregator(t, Config{
		ShardCount: 4,
		Window:     DefaultWindowConfig(),
	})

	events := []Event{
		{Query: "iphone 15", OccurredAt: now.Add(-time.Second)},
		{Query: "iphone 15", OccurredAt: now.Add(-2 * time.Second)},
		{Query: "кроссовки", OccurredAt: now.Add(-time.Second)},
		{Query: "ноутбук", OccurredAt: now.Add(-time.Second)},
		{Query: "ноутбук", OccurredAt: now.Add(-2 * time.Second)},
		{Query: "ноутбук", OccurredAt: now.Add(-3 * time.Second)},
	}

	for _, event := range events {
		result := aggregator.AddAt(event, now)
		if !result.Accepted {
			t.Fatalf("expected event to be accepted, got reason %q", result.Reason)
		}
	}

	top := aggregator.TopAt(10, now)

	want := []Item{
		{Query: "ноутбук", Count: 3},
		{Query: "iphone 15", Count: 2},
		{Query: "кроссовки", Count: 1},
	}

	assertItems(t, top, want)
}

func TestAggregatorTopRespectsLimit(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	aggregator := newTestAggregator(t, Config{
		ShardCount: 4,
		Window:     DefaultWindowConfig(),
	})

	for _, query := range []string{"a", "b", "c", "d"} {
		result := aggregator.AddAt(Event{
			Query:      query,
			OccurredAt: now.Add(-time.Second),
		}, now)

		if !result.Accepted {
			t.Fatalf("expected event to be accepted, got reason %q", result.Reason)
		}
	}

	top := aggregator.TopAt(2, now)
	if len(top) != 2 {
		t.Fatalf("top length mismatch: got %d, want 2", len(top))
	}
}

func TestAggregatorCountAt(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	aggregator := newTestAggregator(t, Config{
		ShardCount: 4,
		Window:     DefaultWindowConfig(),
	})

	for i := 0; i < 3; i++ {
		result := aggregator.AddAt(Event{
			Query:      "iphone 15",
			OccurredAt: now.Add(-time.Duration(i) * time.Second),
		}, now)

		if !result.Accepted {
			t.Fatalf("expected event to be accepted, got reason %q", result.Reason)
		}
	}

	got := aggregator.CountAt("iphone 15", now)
	if got != 3 {
		t.Fatalf("count mismatch: got %d, want 3", got)
	}
}

func TestAggregatorUniqueQueriesAt(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	aggregator := newTestAggregator(t, Config{
		ShardCount: 4,
		Window:     DefaultWindowConfig(),
	})

	events := []Event{
		{Query: "a", OccurredAt: now.Add(-time.Second)},
		{Query: "b", OccurredAt: now.Add(-time.Second)},
		{Query: "b", OccurredAt: now.Add(-2 * time.Second)},
		{Query: "c", OccurredAt: now.Add(-time.Second)},
	}

	for _, event := range events {
		result := aggregator.AddAt(event, now)
		if !result.Accepted {
			t.Fatalf("expected event to be accepted, got reason %q", result.Reason)
		}
	}

	got := aggregator.UniqueQueriesAt(now)
	if got != 3 {
		t.Fatalf("unique query count mismatch: got %d, want 3", got)
	}
}

func TestAggregatorExpiresQueriesAcrossShards(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	aggregator := newTestAggregator(t, Config{
		ShardCount: 4,
		Window:     DefaultWindowConfig(),
	})

	events := []Event{
		{Query: "a", OccurredAt: now.Add(-DefaultWindowSize + time.Second)},
		{Query: "b", OccurredAt: now.Add(-DefaultWindowSize + time.Second)},
	}

	for _, event := range events {
		result := aggregator.AddAt(event, now)
		if !result.Accepted {
			t.Fatalf("expected event to be accepted, got reason %q", result.Reason)
		}
	}

	later := now.Add(2 * time.Second)

	top := aggregator.TopAt(10, later)
	if len(top) != 0 {
		t.Fatalf("expected empty top after expiration, got %#v", top)
	}

	if unique := aggregator.UniqueQueriesAt(later); unique != 0 {
		t.Fatalf("unique query count mismatch after expiration: got %d, want 0", unique)
	}
}

func TestAggregatorPropagatesDropReason(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	aggregator := newTestAggregator(t, Config{
		ShardCount: 4,
		Window:     DefaultWindowConfig(),
	})

	result := aggregator.AddAt(Event{
		Query:      "too old",
		OccurredAt: now.Add(-DefaultWindowSize - time.Second),
	}, now)

	if result.Accepted {
		t.Fatalf("expected event to be rejected")
	}

	if result.Reason != DropReasonTooOld {
		t.Fatalf("reason mismatch: got %q, want %q", result.Reason, DropReasonTooOld)
	}
}

func newTestAggregator(t *testing.T, cfg Config) *Aggregator {
	t.Helper()

	aggregator, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create aggregator: %v", err)
	}

	return aggregator
}
