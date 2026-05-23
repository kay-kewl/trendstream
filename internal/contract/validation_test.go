package contract

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidateAtAcceptsValidEvent(t *testing.T) {
	t.Parallel()

	now := fixedTime()

	event := validEvent(now)
	if err := ValidateAt(event, now); err != nil {
		t.Fatalf("expected valid event, got error: %v", err)
	}
}

func TestValidateAtRejectsUnsupportedSchemaVersion(t *testing.T) {
	t.Parallel()

	now := fixedTime()
	event := validEvent(now)
	event.SchemaVersion = 2

	err := ValidateAt(event, now)
	requireValidationReason(t, err, ValidationReasonUnsupportedSchemaVersion)
}

func TestValidateAtRejectsMissingEventID(t *testing.T) {
	t.Parallel()

	now := fixedTime()
	event := validEvent(now)
	event.EventID = " "

	err := ValidateAt(event, now)
	requireValidationReason(t, err, ValidationReasonMissingEventID)
}

func TestValidateAtRejectsMissingOccurredAt(t *testing.T) {
	t.Parallel()

	now := fixedTime()
	event := validEvent(now)
	event.OccurredAt = time.Time{}

	err := ValidateAt(event, now)
	requireValidationReason(t, err, ValidationReasonMissingOccurredAt)
}

func TestValidateAtRejectsMissingQuery(t *testing.T) {
	t.Parallel()

	now := fixedTime()
	event := validEvent(now)
	event.Query = " \t\n "

	err := ValidateAt(event, now)
	requireValidationReason(t, err, ValidationReasonMissingQuery)
}

func TestValidateAtRejectsTooLongQuery(t *testing.T) {
	t.Parallel()

	now := fixedTime()
	event := validEvent(now)
	event.Query = strings.Repeat("я", MaxQueryRunes+1)

	err := ValidateAt(event, now)
	requireValidationReason(t, err, ValidationReasonQueryTooLong)
}

func TestValidateAtRejectsEventTooFarInFuture(t *testing.T) {
	t.Parallel()

	now := fixedTime()
	event := validEvent(now)
	event.OccurredAt = now.Add(MaxFutureSkew + time.Second)

	err := ValidateAt(event, now)
	requireValidationReason(t, err, ValidationReasonEventFromFuture)
}

func TestSearchEventActorKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event SearchEvent
		want  string
	}{
		{
			name: "user id hash has highest priority",
			event: SearchEvent{
				UserIDHash:   "user",
				DeviceIDHash: "device",
				IPHash:       "ip",
				SessionID:    "session",
			},
			want: "user",
		},
		{
			name: "device id hash is used if user id is missing",
			event: SearchEvent{
				DeviceIDHash: "device",
				IPHash:       "ip",
				SessionID:    "session",
			},
			want: "device",
		},
		{
			name: "ip hash is used if user and device are missing",
			event: SearchEvent{
				IPHash:    "ip",
				SessionID: "session",
			},
			want: "ip",
		},
		{
			name: "session id is used as fallback",
			event: SearchEvent{
				SessionID: "session",
			},
			want: "session",
		},
		{
			name:  "empty actor key if no identity signals are present",
			event: SearchEvent{},
			want:  "",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.event.ActorKey()
			if got != tt.want {
				t.Fatalf("actor key mismatch: got %q, want %q", got, tt.want)
			}
		})
	}
}

func fixedTime() time.Time {
	return time.Date(2026, 5, 23, 9, 0, 0, 0, time.UTC)
}

func validEvent(now time.Time) SearchEvent {
	return SearchEvent{
		SchemaVersion: SearchEventSchemaVersion,
		EventID:       "event-1",
		OccurredAt:    now.Add(-time.Second),
		Query:         "iphone 15",
		UserIDHash:    "user-hash",
	}
}

func requireValidationReason(t *testing.T, err error, reason ValidationReason) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected validation error with reason %q, got nil", reason)
	}

	var validationErr ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	if validationErr.Reason != reason {
		t.Fatalf("validation reason mismatch: got %q, want %q", validationErr.Reason, reason)
	}
}
