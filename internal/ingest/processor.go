package ingest

import (
	"context"
	"errors"
	"time"

	"github.com/kay-kewl/trendstream/internal/aggregator"
	"github.com/kay-kewl/trendstream/internal/contract"
	"github.com/kay-kewl/trendstream/internal/normalize"
)

type StopList interface {
	Contains(rawTerm string) bool
}

type Reason string

const (
	ReasonNone                   Reason = ""
	ReasonInvalidEvent           Reason = "invalid_event"
	ReasonEmptyQuery             Reason = "empty_query"
	ReasonStopList               Reason = "stoplist"
	ReasonBot                    Reason = "bot"
	ReasonTooOld                 Reason = "too_old"
	ReasonFromFuture             Reason = "from_future"
	ReasonCardinalityLimit       Reason = "cardinality_limit"
	ReasonBucketCardinalityLimit Reason = "bucket_cardinality_limit"
	ReasonActorQueryLimit        Reason = "actor_query_limit"
)

type Result struct {
	Accepted bool   `json:"accepted"`
	Reason   Reason `json:"reason,omitempty"`
	Query    string `json:"query,omitempty"`
	Count    int64  `json:"count,omitempty"`
}

type Observer interface {
	ObserveIngestResult(result Result)
}

type Processor struct {
	aggregator *aggregator.Aggregator
	stopList   StopList
	observer   Observer
}

func NewProcessor(aggregator *aggregator.Aggregator, stopList StopList) *Processor {
	return NewProcessorWithObserver(aggregator, stopList, nil)
}

func NewProcessorWithObserver(aggregator *aggregator.Aggregator, stopList StopList, observer Observer) *Processor {
	return &Processor{
		aggregator: aggregator,
		stopList:   stopList,
		observer:   observer,
	}
}

func (p *Processor) Process(ctx context.Context, event contract.SearchEvent) Result {
	return p.ProcessAt(ctx, event, time.Now().UTC())
}

func (p *Processor) ProcessAt(_ context.Context, event contract.SearchEvent, now time.Time) Result {
	if err := contract.ValidateAt(event, now); err != nil {
		return p.finish(Result{
			Accepted: false,
			Reason:   ReasonInvalidEvent,
		})
	}

	query, ok := normalize.Query(event.Query)
	if !ok {
		return p.finish(Result{
			Accepted: false,
			Reason:   ReasonEmptyQuery,
		})
	}

	if event.IsBot {
		return p.finish(Result{
			Accepted: false,
			Reason:   ReasonBot,
			Query:    query,
		})
	}

	if p.stopList != nil && p.stopList.Contains(query) {
		return p.finish(Result{
			Accepted: false,
			Reason:   ReasonStopList,
			Query:    query,
		})
	}

	addResult := p.aggregator.AddAt(aggregator.Event{
		Query:      query,
		OccurredAt: event.OccurredAt,
		ActorKey:   event.ActorKey(),
	}, now)

	if !addResult.Accepted {
		return p.finish(Result{
			Accepted: false,
			Reason:   mapAggregatorDropReason(addResult.Reason),
			Query:    query,
		})
	}

	return p.finish(Result{
		Accepted: true,
		Query:    query,
		Count:    p.aggregator.CountAt(query, now),
	})
}

func (p *Processor) finish(result Result) Result {
	if p.observer != nil {
		p.observer.ObserveIngestResult(result)
	}

	return result
}

func mapAggregatorDropReason(reason aggregator.DropReason) Reason {
	switch reason {
	case aggregator.DropReasonEmptyQuery:
		return ReasonEmptyQuery
	case aggregator.DropReasonTooOld:
		return ReasonTooOld
	case aggregator.DropReasonFromFuture:
		return ReasonFromFuture
	case aggregator.DropReasonCardinalityLimit:
		return ReasonCardinalityLimit
	case aggregator.DropReasonBucketCardinalityLimit:
		return ReasonBucketCardinalityLimit
	case aggregator.DropReasonActorQueryLimit:
		return ReasonActorQueryLimit
	default:
		return ReasonInvalidEvent
	}
}

func IsDropped(result Result) bool {
	return !result.Accepted && result.Reason != ReasonNone
}

func IsInvalid(result Result) bool {
	return errors.Is(reasonError(result.Reason), errInvalidReason)
}

var errInvalidReason = errors.New("invalid result reason")

func reasonError(reason Reason) error {
	switch reason {
	case ReasonInvalidEvent, ReasonEmptyQuery:
		return errInvalidReason
	default:
		return nil
	}
}
