package aggregator

import "time"

type Event struct {
	Query      string
	OccurredAt time.Time
	ActorKey   string
}

type Item struct {
	Query string `json:"query"`
	Count int64  `json:"count"`
}

type DropReason string

const (
	DropReasonNone                   DropReason = ""
	DropReasonEmptyQuery             DropReason = "empty_query"
	DropReasonTooOld                 DropReason = "too_old"
	DropReasonFromFuture             DropReason = "from_future"
	DropReasonCardinalityLimit       DropReason = "cardinality_limit"
	DropReasonBucketCardinalityLimit DropReason = "bucket_cardinality_limit"
	DropReasonActorQueryLimit        DropReason = "actor_query_limit"
)

type AddResult struct {
	Accepted bool
	Reason   DropReason
}
