package ingest

import (
	"context"
	"errors"
	"time"

	"github.com/burtonjake686/trendstream/internal/aggregator"
	"github.com/burtonjake686/trendstream/internal/contract"
	"github.com/burtonjake686/trendstream/internal/normalize"
)

type StopList interface {
	Contains(rawTerm string) bool
}

type Reason string

const (
	ReasonNone         Reason = ""
	ReasonInvalidEvent Reason = "invalid_event"
	ReasonEmptyQuery   Reason = "empty_query"
	ReasonStopList     Reason = "stoplist"
	ReasonBot          Reason = "bot"
	ReasonTooOld       Reason = "too_old"
	ReasonFromFuture   Reason = "from_future"
)

type Result struct {
	Accepted bool   `json:"accepted"`
	Reason   Reason `json:"reason,omitempty"`
	Query    string `json:"query,omitempty"`
	Count    int64  `json:"count,omitempty"`
}

type Processor struct {
	aggregator *aggregator.Aggregator
	stopList   StopList
}

func NewProcessor(aggregator *aggregator.Aggregator, stopList StopList) *Processor {
	return &Processor{
		aggregator: aggregator,
		stopList:   stopList,
	}
}

func (p *Processor) Process(ctx context.Context, event contract.SearchEvent) Result {
	return p.ProcessAt(ctx, event, time.Now().UTC())
}

func (p *Processor) ProcessAt(_ context.Context, event contract.SearchEvent, now time.Time) Result {
	if err := contract.ValidateAt(event, now); err != nil {
		return Result{
			Accepted: false,
			Reason:   ReasonInvalidEvent,
		}
	}

	query, ok := normalize.Query(event.Query)
	if !ok {
		return Result{
			Accepted: false,
			Reason:   ReasonEmptyQuery,
		}
	}

	if event.IsBot {
		return Result{
			Accepted: false,
			Reason:   ReasonBot,
			Query:    query,
		}
	}

	if p.stopList != nil && p.stopList.Contains(query) {
		return Result{
			Accepted: false,
			Reason:   ReasonStopList,
			Query:    query,
		}
	}

	addResult := p.aggregator.AddAt(aggregator.Event{
		Query:      query,
		OccurredAt: event.OccurredAt,
		ActorKey:   event.ActorKey(),
	}, now)

	if !addResult.Accepted {
		return Result{
			Accepted: false,
			Reason:   mapAggregatorDropReason(addResult.Reason),
			Query:    query,
		}
	}

	return Result{
		Accepted: true,
		Query:    query,
		Count:    p.aggregator.CountAt(query, now),
	}
}

func mapAggregatorDropReason(reason aggregator.DropReason) Reason {
	switch reason {
	case aggregator.DropReasonEmptyQuery:
		return ReasonEmptyQuery
	case aggregator.DropReasonTooOld:
		return ReasonTooOld
	case aggregator.DropReasonFromFuture:
		return ReasonFromFuture
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