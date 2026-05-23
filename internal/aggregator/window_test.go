package aggregator

import (
	"testing"
	"time"
)

func TestWindowAddAndTop(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	window := newTestWindow(t)

	result := window.AddAt(Event{
		Query:      "iphone 15",
		OccurredAt: now.Add(-time.Minute),
	}, now)

	if !result.Accepted {
		t.Fatalf("expected event to be accepted, got reason %q", result.Reason)
	}

	top := window.TopAt(10, now)

	if len(top) != 1 {
		t.Fatalf("top length mismatch: got %d, want 1", len(top))
	}

	if top[0].Query != "iphone 15" {
		t.Fatalf("query mismatch: got %q, want %q", top[0].Query, "iphone 15")
	}

	if top[0].Count != 1 {
		t.Fatalf("count mismatch: got %d, want 1", top[0].Count)
	}
}

func TestWindowAggregatesSameQuery(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	window := newTestWindow(t)

	for i := 0; i < 3; i++ {
		result := window.AddAt(Event{
			Query:      "iphone 15",
			OccurredAt: now.Add(-time.Duration(i) * time.Second),
		}, now)

		if !result.Accepted {
			t.Fatalf("expected event %d to be accepted, got reason %q", i, result.Reason)
		}
	}

	count := window.CountAt("iphone 15", now)
	if count != 3 {
		t.Fatalf("count mismatch: got %d, want 3", count)
	}
}

func TestWindowSortsTopByCountThenQuery(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	window := newTestWindow(t)

	events := []Event{
		{Query: "banana", OccurredAt: now.Add(-time.Second)},
		{Query: "banana", OccurredAt: now.Add(-2 * time.Second)},
		{Query: "apple", OccurredAt: now.Add(-time.Second)},
		{Query: "apple", OccurredAt: now.Add(-2 * time.Second)},
		{Query: "phone", OccurredAt: now.Add(-time.Second)},
	}

	for _, event := range events {
		result := window.AddAt(event, now)
		if !result.Accepted {
			t.Fatalf("expected event to be accepted, got reason %q", result.Reason)
		}
	}

	top := window.TopAt(10, now)

	want := []Item{
		{Query: "apple", Count: 2},
		{Query: "banana", Count: 2},
		{Query: "phone", Count: 1},
	}

	assertItems(t, top, want)
}

func TestWindowRespectsLimit(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	window := newTestWindow(t)

	queries := []string{"a", "b", "c"}
	for _, query := range queries {
		result := window.AddAt(Event{
			Query:      query,
			OccurredAt: now.Add(-time.Second),
		}, now)

		if !result.Accepted {
			t.Fatalf("expected event to be accepted, got reason %q", result.Reason)
		}
	}

	top := window.TopAt(2, now)
	if len(top) != 2 {
		t.Fatalf("top length mismatch: got %d, want 2", len(top))
	}
}

func TestWindowExpiresOldBuckets(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	window := newTestWindow(t)

	result := window.AddAt(Event{
		Query:      "old query",
		OccurredAt: now.Add(-DefaultWindowSize + time.Second),
	}, now)

	if !result.Accepted {
		t.Fatalf("expected event to be accepted, got reason %q", result.Reason)
	}

	countBefore := window.CountAt("old query", now)
	if countBefore != 1 {
		t.Fatalf("count before expiration mismatch: got %d, want 1", countBefore)
	}

	later := now.Add(2 * time.Second)

	countAfter := window.CountAt("old query", later)
	if countAfter != 0 {
		t.Fatalf("count after expiration mismatch: got %d, want 0", countAfter)
	}

	if unique := window.UniqueQueriesAt(later); unique != 0 {
		t.Fatalf("unique queries mismatch after expiration: got %d, want 0", unique)
	}
}

func TestWindowAcceptsEventAtWindowBoundary(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	window := newTestWindow(t)

	result := window.AddAt(Event{
		Query:      "boundary query",
		OccurredAt: now.Add(-DefaultWindowSize),
	}, now)

	if !result.Accepted {
		t.Fatalf("expected boundary event to be accepted, got reason %q", result.Reason)
	}

	count := window.CountAt("boundary query", now)
	if count != 1 {
		t.Fatalf("count mismatch: got %d, want 1", count)
	}
}

func TestWindowRejectsTooOldEvent(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	window := newTestWindow(t)

	result := window.AddAt(Event{
		Query:      "too old query",
		OccurredAt: now.Add(-DefaultWindowSize - time.Second),
	}, now)

	if result.Accepted {
		t.Fatalf("expected event to be rejected")
	}

	if result.Reason != DropReasonTooOld {
		t.Fatalf("reason mismatch: got %q, want %q", result.Reason, DropReasonTooOld)
	}
}

func TestWindowRejectsEventTooFarInFuture(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	window := newTestWindow(t)

	result := window.AddAt(Event{
		Query:      "future query",
		OccurredAt: now.Add(DefaultMaxFutureSkew + time.Second),
	}, now)

	if result.Accepted {
		t.Fatalf("expected event to be rejected")
	}

	if result.Reason != DropReasonFromFuture {
		t.Fatalf("reason mismatch: got %q, want %q", result.Reason, DropReasonFromFuture)
	}
}

func TestWindowClampsSmallFutureSkewToNow(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	window := newTestWindow(t)

	result := window.AddAt(Event{
		Query:      "slightly future query",
		OccurredAt: now.Add(DefaultMaxFutureSkew / 2),
	}, now)

	if !result.Accepted {
		t.Fatalf("expected event within future skew to be accepted, got reason %q", result.Reason)
	}

	count := window.CountAt("slightly future query", now)
	if count != 1 {
		t.Fatalf("count mismatch: got %d, want 1", count)
	}
}

func TestWindowRejectsEmptyQuery(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	window := newTestWindow(t)

	result := window.AddAt(Event{
		Query:      " \t\n ",
		OccurredAt: now,
	}, now)

	if result.Accepted {
		t.Fatalf("expected event to be rejected")
	}

	if result.Reason != DropReasonEmptyQuery {
		t.Fatalf("reason mismatch: got %q, want %q", result.Reason, DropReasonEmptyQuery)
	}
}

func TestNewWindowRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  WindowConfig
	}{
		{
			name: "negative window size",
			cfg: WindowConfig{
				WindowSize: -time.Second,
				BucketSize: time.Second,
			},
		},
		{
			name: "negative bucket size",
			cfg: WindowConfig{
				WindowSize: time.Minute,
				BucketSize: -time.Second,
			},
		},
		{
			name: "bucket larger than window",
			cfg: WindowConfig{
				WindowSize: time.Second,
				BucketSize: time.Minute,
			},
		},
		{
			name: "window is not divisible by bucket",
			cfg: WindowConfig{
				WindowSize: 5 * time.Minute,
				BucketSize: 7 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewWindow(tt.cfg)
			if err == nil {
				t.Fatalf("expected error for invalid config")
			}
		})
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 5, 23, 9, 0, 0, 0, time.UTC)
}

func newTestWindow(t *testing.T) *Window {
	t.Helper()

	window, err := NewWindow(DefaultWindowConfig())
	if err != nil {
		t.Fatalf("failed to create window: %v", err)
	}

	return window
}

func assertItems(t *testing.T, got []Item, want []Item) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("items length mismatch: got %d, want %d; got items: %#v", len(got), len(want), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("item %d mismatch: got %#v, want %#v", i, got[i], want[i])
		}
	}
}
