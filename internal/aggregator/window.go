package aggregator

import (
	"errors"
	"sort"
	"strings"
	"time"
)

const (
	DefaultWindowSize    = 5 * time.Minute
	DefaultBucketSize    = time.Second
	DefaultMaxFutureSkew = 10 * time.Second
)

type WindowConfig struct {
	WindowSize    time.Duration
	BucketSize    time.Duration
	MaxFutureSkew time.Duration
}

type bucket struct {
	id     int64
	counts map[string]int64
}

type Window struct {
	cfg     WindowConfig
	buckets []bucket
	totals  map[string]int64
}

func DefaultWindowConfig() WindowConfig {
	return WindowConfig{
		WindowSize:    DefaultWindowSize,
		BucketSize:    DefaultBucketSize,
		MaxFutureSkew: DefaultMaxFutureSkew,
	}
}

func NewWindow(cfg WindowConfig) (*Window, error) {
	cfg = withDefaults(cfg)

	if cfg.WindowSize <= 0 {
		return nil, errors.New("window size must be positive")
	}

	if cfg.BucketSize <= 0 {
		return nil, errors.New("bucket size must be positive")
	}

	if cfg.BucketSize > cfg.WindowSize {
		return nil, errors.New("bucket size must be less than or equal to window size")
	}

	if cfg.WindowSize%cfg.BucketSize != 0 {
		return nil, errors.New("window size must be divisible by bucket size")
	}

	bucketCount := int(cfg.WindowSize/cfg.BucketSize) + 1

	return &Window{
		cfg:     cfg,
		buckets: make([]bucket, bucketCount),
		totals:  make(map[string]int64),
	}, nil
}

func (w *Window) Add(event Event) AddResult {
	return w.AddAt(event, time.Now().UTC())
}

func (w *Window) AddAt(event Event, now time.Time) AddResult {
	query := strings.TrimSpace(event.Query)
	if query == "" {
		return AddResult{
			Accepted: false,
			Reason:   DropReasonEmptyQuery,
		}
	}

	now = now.UTC()
	eventTime := event.OccurredAt.UTC()

	if eventTime.After(now.Add(w.cfg.MaxFutureSkew)) {
		return AddResult{
			Accepted: false,
			Reason:   DropReasonFromFuture,
		}
	}

	if eventTime.After(now) {
		eventTime = now
	}

	w.ExpireAt(now)

	minBucketID := w.minBucketID(now)
	eventBucketID := w.bucketID(eventTime)

	if eventBucketID < minBucketID {
		return AddResult{
			Accepted: false,
			Reason:   DropReasonTooOld,
		}
	}

	targetBucket := w.ensureBucket(eventBucketID)

	targetBucket.counts[query]++
	w.totals[query]++

	return AddResult{
		Accepted: true,
		Reason:   DropReasonNone,
	}
}

func (w *Window) Top(limit int) []Item {
	return w.TopAt(limit, time.Now().UTC())
}

func (w *Window) TopAt(limit int, now time.Time) []Item {
	if limit <= 0 {
		return nil
	}

	w.ExpireAt(now.UTC())

	items := make([]Item, 0, len(w.totals))
	for query, count := range w.totals {
		if count <= 0 {
			continue
		}

		items = append(items, Item{
			Query: query,
			Count: count,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Query < items[j].Query
		}

		return items[i].Count > items[j].Count
	})

	if len(items) > limit {
		items = items[:limit]
	}

	return items
}

func (w *Window) CountAt(query string, now time.Time) int64 {
	w.ExpireAt(now.UTC())

	return w.totals[query]
}

func (w *Window) UniqueQueriesAt(now time.Time) int {
	w.ExpireAt(now.UTC())

	return len(w.totals)
}

func (w *Window) ExpireAt(now time.Time) {
	minBucketID := w.minBucketID(now.UTC())

	for i := range w.buckets {
		current := &w.buckets[i]
		if current.counts == nil {
			continue
		}

		if current.id >= minBucketID {
			continue
		}

		w.subtractBucket(current)
		clear(current.counts)
	}
}

func (w *Window) ensureBucket(bucketID int64) *bucket {
	index := w.bucketIndex(bucketID)
	current := &w.buckets[index]

	if current.counts == nil {
		current.id = bucketID
		current.counts = make(map[string]int64)

		return current
	}

	if current.id != bucketID {
		w.subtractBucket(current)
		clear(current.counts)
		current.id = bucketID
	}

	return current
}

func (w *Window) subtractBucket(current *bucket) {
	for query, count := range current.counts {
		w.totals[query] -= count
		if w.totals[query] <= 0 {
			delete(w.totals, query)
		}
	}
}

func (w *Window) minBucketID(now time.Time) int64 {
	return w.bucketID(now.Add(-w.cfg.WindowSize))
}

func (w *Window) bucketID(ts time.Time) int64 {
	return ts.UTC().UnixNano() / int64(w.cfg.BucketSize)
}

func (w *Window) bucketIndex(bucketID int64) int {
	index := bucketID % int64(len(w.buckets))
	if index < 0 {
		index += int64(len(w.buckets))
	}

	return int(index)
}

func withDefaults(cfg WindowConfig) WindowConfig {
	defaults := DefaultWindowConfig()

	if cfg.WindowSize == 0 {
		cfg.WindowSize = defaults.WindowSize
	}

	if cfg.BucketSize == 0 {
		cfg.BucketSize = defaults.BucketSize
	}

	if cfg.MaxFutureSkew == 0 {
		cfg.MaxFutureSkew = defaults.MaxFutureSkew
	}

	return cfg
}
