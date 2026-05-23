package aggregator

import (
	"errors"
	"sort"
	"time"

	queryhash "github.com/kay-kewl/trendstream/internal/hash"
)

const DefaultShardCount = 32

var ErrInvalidShardCount = errors.New("shard count must be positive")

type Config struct {
	ShardCount int
	Window     WindowConfig
}

type Aggregator struct {
	shards []*Shard
}

func DefaultConfig() Config {
	return Config{
		ShardCount: DefaultShardCount,
		Window:     DefaultWindowConfig(),
	}
}

func New(cfg Config) (*Aggregator, error) {
	cfg = withDefaultAggregatorConfig(cfg)

	if cfg.ShardCount <= 0 {
		return nil, ErrInvalidShardCount
	}

	shards := make([]*Shard, cfg.ShardCount)

	for i := range shards {
		shard, err := NewShard(cfg.Window)
		if err != nil {
			return nil, err
		}

		shards[i] = shard
	}

	return &Aggregator{
		shards: shards,
	}, nil
}

func (a *Aggregator) Add(event Event) AddResult {
	return a.AddAt(event, time.Now().UTC())
}

func (a *Aggregator) AddAt(event Event, now time.Time) AddResult {
	return a.shardFor(event.Query).AddAt(event, now)
}

func (a *Aggregator) Top(limit int) []Item {
	return a.TopAt(limit, time.Now().UTC())
}

func (a *Aggregator) TopAt(limit int, now time.Time) []Item {
	return a.TopFilteredAt(limit, now, nil)
}

func (a *Aggregator) TopFilteredAt(limit int, now time.Time, include func(Item) bool) []Item {
	if limit <= 0 {
		return nil
	}

	candidates := make([]Item, 0, limit*len(a.shards))

	for _, shard := range a.shards {
		candidates = append(candidates, shard.TopFilteredAt(limit, now, include)...)
	}

	sortItems(candidates)

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return candidates
}

func (a *Aggregator) CountAt(query string, now time.Time) int64 {
	return a.shardFor(query).CountAt(query, now)
}

func (a *Aggregator) UniqueQueriesAt(now time.Time) int {
	total := 0

	for _, shard := range a.shards {
		total += shard.UniqueQueriesAt(now)
	}

	return total
}

func (a *Aggregator) ActorCountersAt(now time.Time) int {
	total := 0

	for _, shard := range a.shards {
		total += shard.ActorCountersAt(now)
	}

	return total
}

func (a *Aggregator) WindowEventsAt(now time.Time) int64 {
	var total int64

	for _, shard := range a.shards {
		total += shard.WindowEventsAt(now)
	}

	return total
}

func (a *Aggregator) ShardCount() int {
	return len(a.shards)
}

func (a *Aggregator) ShardIndex(query string) int {
	return queryhash.Index(query, len(a.shards))
}

func (a *Aggregator) shardFor(query string) *Shard {
	return a.shards[a.ShardIndex(query)]
}

func sortItems(items []Item) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Query < items[j].Query
		}

		return items[i].Count > items[j].Count
	})
}

func withDefaultAggregatorConfig(cfg Config) Config {
	defaults := DefaultConfig()

	if cfg.ShardCount == 0 {
		cfg.ShardCount = defaults.ShardCount
	}

	if cfg.Window.WindowSize == 0 {
		cfg.Window.WindowSize = defaults.Window.WindowSize
	}

	if cfg.Window.BucketSize == 0 {
		cfg.Window.BucketSize = defaults.Window.BucketSize
	}

	if cfg.Window.MaxFutureSkew == 0 {
		cfg.Window.MaxFutureSkew = defaults.Window.MaxFutureSkew
	}

	if cfg.Window.MaxUniqueQueries == 0 {
		cfg.Window.MaxUniqueQueries = defaults.Window.MaxUniqueQueries
	}

	if cfg.Window.MaxUniqueQueriesPerBucket == 0 {
		cfg.Window.MaxUniqueQueriesPerBucket = defaults.Window.MaxUniqueQueriesPerBucket
	}

	if cfg.Window.PerActorQueryLimit == 0 {
		cfg.Window.PerActorQueryLimit = defaults.Window.PerActorQueryLimit
	}

	return cfg
}
