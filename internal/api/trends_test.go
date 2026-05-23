package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kay-kewl/trendstream/internal/aggregator"
	"github.com/kay-kewl/trendstream/internal/snapshot"
)

func TestGetTrendsReturnsDefaultLimit(t *testing.T) {
	t.Parallel()

	handler := newTestTrendsHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/v1/trends", nil)
	recorder := httptest.NewRecorder()

	handler.GetTrends(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response snapshot.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response.Items) != 2 {
		t.Fatalf("items length mismatch: got %d, want 2", len(response.Items))
	}
}

func TestGetTrendsRespectsLimit(t *testing.T) {
	t.Parallel()

	handler := newTestTrendsHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/v1/trends?limit=1", nil)
	recorder := httptest.NewRecorder()

	handler.GetTrends(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response snapshot.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response.Items) != 1 {
		t.Fatalf("items length mismatch: got %d, want 1", len(response.Items))
	}

	if response.Items[0].Query != "iphone" {
		t.Fatalf("query mismatch: got %q, want %q", response.Items[0].Query, "iphone")
	}
}

func TestGetTrendsRejectsNonIntegerLimit(t *testing.T) {
	t.Parallel()

	handler := newTestTrendsHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/v1/trends?limit=abc", nil)
	recorder := httptest.NewRecorder()

	handler.GetTrends(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestGetTrendsRejectsZeroLimit(t *testing.T) {
	t.Parallel()

	handler := newTestTrendsHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/v1/trends?limit=0", nil)
	recorder := httptest.NewRecorder()

	handler.GetTrends(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestGetTrendsRejectsTooLargeLimit(t *testing.T) {
	t.Parallel()

	handler := newTestTrendsHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/v1/trends?limit=101", nil)
	recorder := httptest.NewRecorder()

	handler.GetTrends(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestGetTrendsReturnsJSONContentType(t *testing.T) {
	t.Parallel()

	handler := newTestTrendsHandler(t)

	request := httptest.NewRequest(http.MethodGet, "/v1/trends?limit=20", nil)
	recorder := httptest.NewRecorder()

	handler.GetTrends(recorder, request)

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Fatalf("content type mismatch: got %q, want %q", contentType, "application/json; charset=utf-8")
	}
}

func newTestTrendsHandler(t *testing.T) *TrendsHandler {
	t.Helper()

	snap, err := snapshot.New([]aggregator.Item{
		{Query: "iphone", Count: 10},
		{Query: "кроссовки", Count: 5},
	}, time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC), snapshot.DefaultOptions())
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	return NewTrendsHandler(snapshot.NewPublisher(snap))
}
