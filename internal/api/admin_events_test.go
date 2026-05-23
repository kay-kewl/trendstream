package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kay-kewl/trendstream/internal/auth"
	"github.com/kay-kewl/trendstream/internal/contract"
	"github.com/kay-kewl/trendstream/internal/ingest"
)

type fakeAdminEventProcessor struct {
	result ingest.Result
}

func (p fakeAdminEventProcessor) ProcessHTTP(r *http.Request, event contract.SearchEvent) ingest.Result {
	return p.result
}

func TestAdminEventsAcceptsValidEvent(t *testing.T) {
	t.Parallel()

	mux := newAdminEventsTestMux(fakeAdminEventProcessor{
		result: ingest.Result{
			Accepted: true,
			Query:    "iphone",
			Count:    1,
		},
	})

	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/events",
		bytes.NewBufferString(`{
			"schema_version": 1,
			"event_id": "event-1",
			"occurred_at": "2026-05-23T12:00:00Z",
			"query": "iphone"
		}`),
	)
	request.Header.Set("Authorization", "Bearer token")

	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status mismatch: got %d, want %d; body=%s", recorder.Code, http.StatusAccepted, recorder.Body.String())
	}
}

func TestAdminEventsRejectsUnauthorizedRequest(t *testing.T) {
	t.Parallel()

	mux := newAdminEventsTestMux(fakeAdminEventProcessor{})

	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/events",
		bytes.NewBufferString(`{}`),
	)

	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestAdminEventsRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	mux := newAdminEventsTestMux(fakeAdminEventProcessor{})

	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/events",
		bytes.NewBufferString(`{invalid json`),
	)
	request.Header.Set("Authorization", "Bearer token")

	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestAdminEventsReturnsBadRequestForInvalidEvent(t *testing.T) {
	t.Parallel()

	mux := newAdminEventsTestMux(fakeAdminEventProcessor{
		result: ingest.Result{
			Accepted: false,
			Reason:   ingest.ReasonInvalidEvent,
		},
	})

	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/events",
		bytes.NewBufferString(`{
			"schema_version": 1,
			"event_id": "",
			"occurred_at": "2026-05-23T12:00:00Z",
			"query": "iphone"
		}`),
	)
	request.Header.Set("Authorization", "Bearer token")

	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestAdminEventsReturnsOKForDroppedEvent(t *testing.T) {
	t.Parallel()

	mux := newAdminEventsTestMux(fakeAdminEventProcessor{
		result: ingest.Result{
			Accepted: false,
			Reason:   ingest.ReasonStopList,
			Query:    "casino",
		},
	})

	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/events",
		bytes.NewBufferString(`{
			"schema_version": 1,
			"event_id": "event-1",
			"occurred_at": "2026-05-23T12:00:00Z",
			"query": "casino"
		}`),
	)
	request.Header.Set("Authorization", "Bearer token")

	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
}

func newAdminEventsTestMux(processor AdminEventProcessor) *http.ServeMux {
	mux := http.NewServeMux()

	handler := NewAdminEventsHandler(processor, auth.NewTokenAuth("token"))
	handler.Register(mux)

	return mux
}
