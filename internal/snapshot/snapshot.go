package snapshot

import (
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/kay-kewl/trendstream/internal/aggregator"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100

	DefaultWindowSeconds = 300
)

var (
	ErrInvalidLimit        = errors.New("limit must be positive")
	ErrLimitTooLarge       = errors.New("limit is too large")
	ErrInvalidWindowSecond = errors.New("window seconds must be positive")
)

type Options struct {
	WindowSeconds     int
	MaxLimit          int
	PrecomputedLimits []int
}

type Snapshot struct {
	GeneratedAt   time.Time
	WindowSeconds int
	Items         []aggregator.Item

	jsonByLimit map[int][]byte
}

type Response struct {
	WindowSeconds int               `json:"window_seconds"`
	GeneratedAt   time.Time         `json:"generated_at"`
	Items         []aggregator.Item `json:"items"`
}

func DefaultOptions() Options {
	return Options{
		WindowSeconds:     DefaultWindowSeconds,
		MaxLimit:          MaxLimit,
		PrecomputedLimits: []int{10, 20, 50, 100},
	}
}

func New(items []aggregator.Item, generatedAt time.Time, opts Options) (*Snapshot, error) {
	opts = withDefaultOptions(opts)

	if opts.WindowSeconds <= 0 {
		return nil, ErrInvalidWindowSecond
	}

	if opts.MaxLimit <= 0 {
		return nil, ErrInvalidLimit
	}

	copiedItems := copyItems(items)
	if len(copiedItems) > opts.MaxLimit {
		copiedItems = copiedItems[:opts.MaxLimit]
	}

	snapshot := &Snapshot{
		GeneratedAt:   generatedAt.UTC(),
		WindowSeconds: opts.WindowSeconds,
		Items:         copiedItems,
		jsonByLimit:   make(map[int][]byte, len(opts.PrecomputedLimits)),
	}

	for _, limit := range opts.PrecomputedLimits {
		if limit <= 0 || limit > opts.MaxLimit {
			continue
		}

		payload, err := snapshot.MarshalLimit(limit)
		if err != nil {
			return nil, err
		}

		snapshot.jsonByLimit[limit] = payload
	}

	return snapshot, nil
}

func Empty(generatedAt time.Time) *Snapshot {
	snapshot, err := New(nil, generatedAt, DefaultOptions())
	if err != nil {
		panic(err)
	}

	return snapshot
}

func (s *Snapshot) Response(limit int) (Response, error) {
	if err := ValidateLimit(limit); err != nil {
		return Response{}, err
	}

	items := s.itemsForLimit(limit)

	return Response{
		WindowSeconds: s.WindowSeconds,
		GeneratedAt:   s.GeneratedAt,
		Items:         items,
	}, nil
}

func (s *Snapshot) MarshalLimit(limit int) ([]byte, error) {
	response, err := s.Response(limit)
	if err != nil {
		return nil, err
	}

	return json.Marshal(response)
}

func (s *Snapshot) PrecomputedJSON(limit int) ([]byte, bool) {
	payload, ok := s.jsonByLimit[limit]
	if !ok {
		return nil, false
	}

	return slices.Clone(payload), true
}

func (s *Snapshot) itemsForLimit(limit int) []aggregator.Item {
	if len(s.Items) == 0 {
		return []aggregator.Item{}
	}

	if limit > len(s.Items) {
		limit = len(s.Items)
	}

	return copyItems(s.Items[:limit])
}

func ValidateLimit(limit int) error {
	if limit <= 0 {
		return ErrInvalidLimit
	}

	if limit > MaxLimit {
		return ErrLimitTooLarge
	}

	return nil
}

func copyItems(items []aggregator.Item) []aggregator.Item {
	if len(items) == 0 {
		return nil
	}

	return slices.Clone(items)
}

func withDefaultOptions(opts Options) Options {
	defaults := DefaultOptions()

	if opts.WindowSeconds == 0 {
		opts.WindowSeconds = defaults.WindowSeconds
	}

	if opts.MaxLimit == 0 {
		opts.MaxLimit = defaults.MaxLimit
	}

	if len(opts.PrecomputedLimits) == 0 {
		opts.PrecomputedLimits = defaults.PrecomputedLimits
	}

	return opts
}
