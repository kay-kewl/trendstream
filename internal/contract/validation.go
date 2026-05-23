package contract

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxQueryRunes = 256
	MaxFutureSkew = 10 * time.Second
)

type ValidationReason string

const (
	ValidationReasonUnsupportedSchemaVersion ValidationReason = "unsupported_schema_version"
	ValidationReasonMissingEventID           ValidationReason = "missing_event_id"
	ValidationReasonMissingOccurredAt        ValidationReason = "missing_occurred_at"
	ValidationReasonMissingQuery             ValidationReason = "missing_query"
	ValidationReasonQueryTooLong             ValidationReason = "query_too_long"
	ValidationReasonEventFromFuture          ValidationReason = "event_from_future"
)

type ValidationError struct {
	Reason  ValidationReason
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

func Validate(event SearchEvent) error {
	return ValidateAt(event, time.Now().UTC())
}

func ValidateAt(event SearchEvent, now time.Time) error {
	if event.SchemaVersion != SearchEventSchemaVersion {
		return ValidationError{
			Reason: ValidationReasonUnsupportedSchemaVersion,
			Message: fmt.Sprintf(
				"unsupported schema version: got %d, want %d",
				event.SchemaVersion,
				SearchEventSchemaVersion,
			),
		}
	}

	if strings.TrimSpace(event.EventID) == "" {
		return ValidationError{
			Reason:  ValidationReasonMissingEventID,
			Message: "event_id is required",
		}
	}

	if event.OccurredAt.IsZero() {
		return ValidationError{
			Reason:  ValidationReasonMissingOccurredAt,
			Message: "occurred_at is required",
		}
	}

	if strings.TrimSpace(event.Query) == "" {
		return ValidationError{
			Reason:  ValidationReasonMissingQuery,
			Message: "query is required",
		}
	}

	if utf8.RuneCountInString(strings.TrimSpace(event.Query)) > MaxQueryRunes {
		return ValidationError{
			Reason: ValidationReasonQueryTooLong,
			Message: fmt.Sprintf(
				"query is too long: max %d runes",
				MaxQueryRunes,
			),
		}
	}

	if event.OccurredAt.After(now.Add(MaxFutureSkew)) {
		return ValidationError{
			Reason: ValidationReasonEventFromFuture,
			Message: fmt.Sprintf(
				"occurred_at is too far in the future: max skew is %s",
				MaxFutureSkew,
			),
		}
	}

	return nil
}
