package snapshot

import (
	"testing"
	"time"

	"github.com/kay-kewl/trendstream/internal/aggregator"
)

func TestNewPublisherUsesInitialSnapshot(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)

	initial, err := New([]aggregator.Item{
		{Query: "initial", Count: 1},
	}, now, DefaultOptions())
	if err != nil {
		t.Fatalf("failed to create initial snapshot: %v", err)
	}

	publisher := NewPublisher(initial)

	current := publisher.Current()
	if current.GeneratedAt != now {
		t.Fatalf("generated_at mismatch: got %s, want %s", current.GeneratedAt, now)
	}

	if len(current.Items) != 1 || current.Items[0].Query != "initial" {
		t.Fatalf("unexpected current snapshot: %#v", current.Items)
	}
}

func TestPublisherPublishReplacesSnapshot(t *testing.T) {
	t.Parallel()

	first, err := New([]aggregator.Item{
		{Query: "first", Count: 1},
	}, time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC), DefaultOptions())
	if err != nil {
		t.Fatalf("failed to create first snapshot: %v", err)
	}

	second, err := New([]aggregator.Item{
		{Query: "second", Count: 2},
	}, time.Date(2026, 5, 23, 12, 1, 0, 0, time.UTC), DefaultOptions())
	if err != nil {
		t.Fatalf("failed to create second snapshot: %v", err)
	}

	publisher := NewPublisher(first)
	publisher.Publish(second)

	current := publisher.Current()
	if len(current.Items) != 1 {
		t.Fatalf("items length mismatch: got %d, want 1", len(current.Items))
	}

	if current.Items[0].Query != "second" {
		t.Fatalf("query mismatch: got %q, want %q", current.Items[0].Query, "second")
	}
}

func TestPublisherIgnoresNilSnapshot(t *testing.T) {
	t.Parallel()

	initial, err := New([]aggregator.Item{
		{Query: "initial", Count: 1},
	}, time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC), DefaultOptions())
	if err != nil {
		t.Fatalf("failed to create initial snapshot: %v", err)
	}

	publisher := NewPublisher(initial)
	publisher.Publish(nil)

	current := publisher.Current()
	if len(current.Items) != 1 || current.Items[0].Query != "initial" {
		t.Fatalf("nil snapshot should be ignored, got %#v", current.Items)
	}
}

func TestNewPublisherWithNilInitialSnapshotCreatesEmptySnapshot(t *testing.T) {
	t.Parallel()

	publisher := NewPublisher(nil)

	current := publisher.Current()
	if current == nil {
		t.Fatalf("current snapshot should not be nil")
	}

	if len(current.Items) != 0 {
		t.Fatalf("expected empty snapshot, got %#v", current.Items)
	}
}
