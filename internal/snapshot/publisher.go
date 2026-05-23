package snapshot

import (
	"sync/atomic"
	"time"
)

type Publisher struct {
	current atomic.Pointer[Snapshot]
}

func NewPublisher(initial *Snapshot) *Publisher {
	publisher := &Publisher{}

	if initial == nil {
		initial = Empty(time.Now().UTC())
	}

	publisher.Publish(initial)

	return publisher
}

func (p *Publisher) Publish(snapshot *Snapshot) {
	if snapshot == nil {
		return
	}

	p.current.Store(snapshot)
}

func (p *Publisher) Current() *Snapshot {
	current := p.current.Load()
	if current == nil {
		return Empty(time.Now().UTC())
	}

	return current
}
