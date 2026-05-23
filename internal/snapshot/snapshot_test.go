package snapshot

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/kay-kewl/trendstream/internal/aggregator"
)

func TestNewCreatesImmutableCopyOfItems(t *testing.T) {
	t.Parallel()

	now := fixedSnapshotTime()

	items := []aggregator.Item{
		{Query: "iphone", Count: 10},
	}

	snap, err := New(items, now, DefaultOptions())
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	items[0].Count = 100

	response, err := snap.Response(10)
	if err != nil {
		t.Fatalf("failed to build response: %v", err)
	}

	if response.Items[0].Count != 10 {
		t.Fatalf("snapshot should not be affected by source slice mutation: got %d, want 10", response.Items[0].Count)
	}
}

func TestSnapshotResponseRespectsLimit(t *testing.T) {
	t.Parallel()

	now := fixedSnapshotTime()

	snap, err := New([]aggregator.Item{
		{Query: "a", Count: 3},
		{Query: "b", Count: 2},
		{Query: "c", Count: 1},
	}, now, DefaultOptions())
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	response, err := snap.Response(2)
	if err != nil {
		t.Fatalf("failed to build response: %v", err)
	}

	if len(response.Items) != 2 {
		t.Fatalf("items length mismatch: got %d, want 2", len(response.Items))
	}

	if response.Items[0].Query != "a" || response.Items[1].Query != "b" {
		t.Fatalf("unexpected response items: %#v", response.Items)
	}
}

func TestSnapshotResponseReturnsEmptyItemsForEmptySnapshot(t *testing.T) {
	t.Parallel()

	snap := Empty(fixedSnapshotTime())

	response, err := snap.Response(20)
	if err != nil {
		t.Fatalf("failed to build response: %v", err)
	}

	if response.Items == nil {
		t.Fatalf("items should be an empty slice, not nil")
	}

	if len(response.Items) != 0 {
		t.Fatalf("items length mismatch: got %d, want 0", len(response.Items))
	}
}

func TestSnapshotResponseRejectsInvalidLimit(t *testing.T) {
	t.Parallel()

	snap := Empty(fixedSnapshotTime())

	_, err := snap.Response(0)
	if !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("expected ErrInvalidLimit, got %v", err)
	}

	_, err = snap.Response(MaxLimit + 1)
	if !errors.Is(err, ErrLimitTooLarge) {
		t.Fatalf("expected ErrLimitTooLarge, got %v", err)
	}
}

func TestPrecomputedJSON(t *testing.T) {
	t.Parallel()

	now := fixedSnapshotTime()

	snap, err := New([]aggregator.Item{
		{Query: "iphone", Count: 10},
	}, now, Options{
		WindowSeconds:     300,
		MaxLimit:          100,
		PrecomputedLimits: []int{20},
	})
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	payload, ok := snap.PrecomputedJSON(20)
	if !ok {
		t.Fatalf("expected precomputed json for limit 20")
	}

	var response Response
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("failed to unmarshal precomputed json: %v", err)
	}

	if response.WindowSeconds != 300 {
		t.Fatalf("window seconds mismatch: got %d, want 300", response.WindowSeconds)
	}

	if len(response.Items) != 1 {
		t.Fatalf("items length mismatch: got %d, want 1", len(response.Items))
	}

	if response.Items[0].Query != "iphone" {
		t.Fatalf("query mismatch: got %q, want %q", response.Items[0].Query, "iphone")
	}
}

func TestPrecomputedJSONReturnsCopy(t *testing.T) {
	t.Parallel()

	snap := Empty(fixedSnapshotTime())

	first, ok := snap.PrecomputedJSON(20)
	if !ok {
		t.Fatalf("expected precomputed json")
	}

	first[0] = '{'

	second, ok := snap.PrecomputedJSON(20)
	if !ok {
		t.Fatalf("expected precomputed json")
	}

	if len(second) == 0 {
		t.Fatalf("unexpected empty json")
	}

	if second[0] != '{' {
		t.Fatalf("expected original json payload to remain valid")
	}
}

func TestNewRejectsInvalidOptions(t *testing.T) {
	t.Parallel()

	_, err := New(nil, fixedSnapshotTime(), Options{
		WindowSeconds: -1,
		MaxLimit:      100,
	})
	if !errors.Is(err, ErrInvalidWindowSecond) {
		t.Fatalf("expected ErrInvalidWindowSecond, got %v", err)
	}

	_, err = New(nil, fixedSnapshotTime(), Options{
		WindowSeconds: 300,
		MaxLimit:      -1,
	})
	if !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("expected ErrInvalidLimit, got %v", err)
	}
}

func fixedSnapshotTime() time.Time {
	return time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
}