package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/kay-kewl/trendstream/internal/aggregator"
	"github.com/kay-kewl/trendstream/internal/contract"
)

type fakeStopList struct {
	blocked map[string]struct{}
}

func (s fakeStopList) Contains(rawTerm string) bool {
	_, exists := s.blocked[rawTerm]
	return exists
}

func TestProcessorAcceptsValidEvent(t *testing.T) {
	t.Parallel()

	now := fixedProcessorTime()
	processor := newTestProcessor(t, fakeStopList{})

	result := processor.ProcessAt(context.Background(), validSearchEvent(now), now)

	if !result.Accepted {
		t.Fatalf("expected event to be accepted, got reason %q", result.Reason)
	}

	if result.Query != "iphone 15 pro" {
		t.Fatalf("query mismatch: got %q, want %q", result.Query, "iphone 15 pro")
	}

	if result.Count != 1 {
		t.Fatalf("count mismatch: got %d, want 1", result.Count)
	}
}

func TestProcessorNormalizesAndAggregatesQueries(t *testing.T) {
	t.Parallel()

	now := fixedProcessorTime()
	processor := newTestProcessor(t, fakeStopList{})

	first := validSearchEvent(now)
	first.Query = "  IPhone   15 PRO "

	second := validSearchEvent(now)
	second.EventID = "event-2"
	second.Query = "iphone 15 pro"

	result := processor.ProcessAt(context.Background(), first, now)
	if !result.Accepted {
		t.Fatalf("expected first event to be accepted, got reason %q", result.Reason)
	}

	result = processor.ProcessAt(context.Background(), second, now)
	if !result.Accepted {
		t.Fatalf("expected second event to be accepted, got reason %q", result.Reason)
	}

	if result.Count != 2 {
		t.Fatalf("count mismatch: got %d, want 2", result.Count)
	}
}

func TestProcessorRejectsInvalidEvent(t *testing.T) {
	t.Parallel()

	now := fixedProcessorTime()
	processor := newTestProcessor(t, fakeStopList{})

	event := validSearchEvent(now)
	event.EventID = ""

	result := processor.ProcessAt(context.Background(), event, now)

	if result.Accepted {
		t.Fatalf("expected invalid event to be rejected")
	}

	if result.Reason != ReasonInvalidEvent {
		t.Fatalf("reason mismatch: got %q, want %q", result.Reason, ReasonInvalidEvent)
	}
}

func TestProcessorRejectsBotEvent(t *testing.T) {
	t.Parallel()

	now := fixedProcessorTime()
	processor := newTestProcessor(t, fakeStopList{})

	event := validSearchEvent(now)
	event.IsBot = true

	result := processor.ProcessAt(context.Background(), event, now)

	if result.Accepted {
		t.Fatalf("expected bot event to be rejected")
	}

	if result.Reason != ReasonBot {
		t.Fatalf("reason mismatch: got %q, want %q", result.Reason, ReasonBot)
	}

	if result.Query != "iphone 15 pro" {
		t.Fatalf("query mismatch: got %q, want %q", result.Query, "iphone 15 pro")
	}
}

func TestProcessorRejectsStopListedQuery(t *testing.T) {
	t.Parallel()

	now := fixedProcessorTime()
	processor := newTestProcessor(t, fakeStopList{
		blocked: map[string]struct{}{
			"iphone 15 pro": {},
		},
	})

	result := processor.ProcessAt(context.Background(), validSearchEvent(now), now)

	if result.Accepted {
		t.Fatalf("expected stop-listed event to be rejected")
	}

	if result.Reason != ReasonStopList {
		t.Fatalf("reason mismatch: got %q, want %q", result.Reason, ReasonStopList)
	}

	if result.Query != "iphone 15 pro" {
		t.Fatalf("query mismatch: got %q, want %q", result.Query, "iphone 15 pro")
	}
}

func TestProcessorRejectsTooOldEvent(t *testing.T) {
	t.Parallel()

	now := fixedProcessorTime()
	processor := newTestProcessor(t, fakeStopList{})

	event := validSearchEvent(now)
	event.OccurredAt = now.Add(-aggregator.DefaultWindowSize - time.Second)

	result := processor.ProcessAt(context.Background(), event, now)

	if result.Accepted {
		t.Fatalf("expected too old event to be rejected")
	}

	if result.Reason != ReasonTooOld {
		t.Fatalf("reason mismatch: got %q, want %q", result.Reason, ReasonTooOld)
	}
}

func TestProcessorRejectsFutureEvent(t *testing.T) {
	t.Parallel()

	now := fixedProcessorTime()
	processor := newTestProcessor(t, fakeStopList{})

	event := validSearchEvent(now)
	event.OccurredAt = now.Add(contract.MaxFutureSkew + time.Second)

	result := processor.ProcessAt(context.Background(), event, now)

	if result.Accepted {
		t.Fatalf("expected future event to be rejected")
	}

	if result.Reason != ReasonInvalidEvent {
		t.Fatalf("reason mismatch: got %q, want %q", result.Reason, ReasonInvalidEvent)
	}
}

func TestIsDropped(t *testing.T) {
	t.Parallel()

	if !IsDropped(Result{Accepted: false, Reason: ReasonStopList}) {
		t.Fatalf("expected result to be dropped")
	}

	if IsDropped(Result{Accepted: true}) {
		t.Fatalf("accepted result should not be dropped")
	}
}

func fixedProcessorTime() time.Time {
	return time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
}

func validSearchEvent(now time.Time) contract.SearchEvent {
	return contract.SearchEvent{
		SchemaVersion: contract.SearchEventSchemaVersion,
		EventID:       "event-1",
		OccurredAt:    now.Add(-time.Second),
		Query:         "  IPhone   15 PRO ",
		UserIDHash:    "user-hash",
	}
}

func newTestProcessor(t *testing.T, stopList StopList) *Processor {
	t.Helper()

	trendAggregator, err := aggregator.New(aggregator.Config{
		ShardCount: 4,
		Window:     aggregator.DefaultWindowConfig(),
	})
	if err != nil {
		t.Fatalf("failed to create aggregator: %v", err)
	}

	return NewProcessor(trendAggregator, stopList)
}

func TestProcessorRejectsActorQueryLimit(t *testing.T) {
	t.Parallel()

	now := fixedProcessorTime()

	trendAggregator, err := aggregator.New(aggregator.Config{
		ShardCount: 4,
		Window: aggregator.WindowConfig{
			WindowSize:                aggregator.DefaultWindowSize,
			BucketSize:                aggregator.DefaultBucketSize,
			MaxFutureSkew:             aggregator.DefaultMaxFutureSkew,
			MaxUniqueQueries:          100,
			MaxUniqueQueriesPerBucket: 100,
			PerActorQueryLimit:        2,
		},
	})
	if err != nil {
		t.Fatalf("failed to create aggregator: %v", err)
	}

	processor := NewProcessor(trendAggregator, fakeStopList{})

	for i := 0; i < 2; i++ {
		event := validSearchEvent(now)
		event.EventID = "event-accepted-" + string(rune('0'+i))
		event.UserIDHash = "same-actor"

		result := processor.ProcessAt(context.Background(), event, now)
		if !result.Accepted {
			t.Fatalf("expected event %d to be accepted, got reason %q", i, result.Reason)
		}
	}

	event := validSearchEvent(now)
	event.EventID = "event-rejected"
	event.UserIDHash = "same-actor"

	result := processor.ProcessAt(context.Background(), event, now)
	if result.Accepted {
		t.Fatalf("expected event over actor limit to be rejected")
	}

	if result.Reason != ReasonActorQueryLimit {
		t.Fatalf("reason mismatch: got %q, want %q", result.Reason, ReasonActorQueryLimit)
	}
}
