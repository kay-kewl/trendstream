package kafka

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/kay-kewl/trendstream/internal/contract"
)

var (
	ErrEmptyPayload       = errors.New("empty kafka message payload")
	ErrMultipleJSONValues = errors.New("payload contains multiple json values")
)

func DecodeSearchEvent(payload []byte) (contract.SearchEvent, error) {
	if len(bytes.TrimSpace(payload)) == 0 {
		return contract.SearchEvent{}, ErrEmptyPayload
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))

	var event contract.SearchEvent
	if err := decoder.Decode(&event); err != nil {
		return contract.SearchEvent{}, fmt.Errorf("decode search event: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != nil && !errors.Is(err, io.EOF) {
		return contract.SearchEvent{}, fmt.Errorf("decode trailing payload: %w", err)
	} else if err == nil {
		return contract.SearchEvent{}, ErrMultipleJSONValues
	}

	return event, nil
}
