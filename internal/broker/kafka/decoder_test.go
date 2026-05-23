package kafka

import (
	"errors"
	"testing"

	"github.com/kay-kewl/trendstream/internal/contract"
)

func TestDecodeSearchEvent(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"schema_version": 1,
		"event_id": "event-1",
		"occurred_at": "2026-05-23T12:00:00Z",
		"query": "iphone 15",
		"user_id_hash": "user-1"
	}`)

	event, err := DecodeSearchEvent(payload)
	if err != nil {
		t.Fatalf("failed to decode event: %v", err)
	}

	if event.SchemaVersion != contract.SearchEventSchemaVersion {
		t.Fatalf("schema version mismatch: got %d, want %d", event.SchemaVersion, contract.SearchEventSchemaVersion)
	}

	if event.EventID != "event-1" {
		t.Fatalf("event id mismatch: got %q, want %q", event.EventID, "event-1")
	}

	if event.Query != "iphone 15" {
		t.Fatalf("query mismatch: got %q, want %q", event.Query, "iphone 15")
	}

	if event.UserIDHash != "user-1" {
		t.Fatalf("user id hash mismatch: got %q, want %q", event.UserIDHash, "user-1")
	}
}

func TestDecodeSearchEventRejectsEmptyPayload(t *testing.T) {
	t.Parallel()

	_, err := DecodeSearchEvent([]byte(" \t\n "))
	if !errors.Is(err, ErrEmptyPayload) {
		t.Fatalf("expected ErrEmptyPayload, got %v", err)
	}
}

func TestDecodeSearchEventRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := DecodeSearchEvent([]byte(`{invalid json`))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDecodeSearchEventRejectsMultipleJSONValues(t *testing.T) {
	t.Parallel()

	_, err := DecodeSearchEvent([]byte(`{"schema_version":1} {"schema_version":1}`))
	if !errors.Is(err, ErrMultipleJSONValues) {
		t.Fatalf("expected ErrMultipleJSONValues, got %v", err)
	}
}
