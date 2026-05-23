package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kay-kewl/trendstream/internal/auth"
	"github.com/kay-kewl/trendstream/internal/stoplist"
)

type fakeStopListService struct {
	terms      []string
	addTerm    string
	addChanged bool
	addErr     error

	removeTerm    string
	removeChanged bool
	removeErr     error
}

func (s *fakeStopListService) Terms() []string {
	return append([]string(nil), s.terms...)
}

func (s *fakeStopListService) Add(rawTerm string) (string, bool, error) {
	if s.addErr != nil {
		return "", false, s.addErr
	}

	return s.addTerm, s.addChanged, nil
}

func (s *fakeStopListService) Remove(rawTerm string) (string, bool, error) {
	if s.removeErr != nil {
		return "", false, s.removeErr
	}

	return s.removeTerm, s.removeChanged, nil
}

func TestAdminStopListList(t *testing.T) {
	t.Parallel()

	service := &fakeStopListService{
		terms: []string{"casino", "spam"},
	}

	mux := newStopListTestMux(service)

	request := httptest.NewRequest(http.MethodGet, "/admin/stop-list", nil)
	request.Header.Set("Authorization", "Bearer token")

	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response stopListResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response.Terms) != 2 {
		t.Fatalf("terms length mismatch: got %d, want 2", len(response.Terms))
	}
}

func TestAdminStopListRejectsUnauthorizedRequest(t *testing.T) {
	t.Parallel()

	mux := newStopListTestMux(&fakeStopListService{})

	request := httptest.NewRequest(http.MethodGet, "/admin/stop-list", nil)
	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestAdminStopListRejectsForbiddenRequest(t *testing.T) {
	t.Parallel()

	mux := newStopListTestMux(&fakeStopListService{})

	request := httptest.NewRequest(http.MethodGet, "/admin/stop-list", nil)
	request.Header.Set("Authorization", "Bearer wrong")

	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestAdminStopListAdd(t *testing.T) {
	t.Parallel()

	service := &fakeStopListService{
		addTerm:    "casino online",
		addChanged: true,
	}

	mux := newStopListTestMux(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/stop-list",
		bytes.NewBufferString(`{"term":" Casino   Online "}`),
	)
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status mismatch: got %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response stopListTermResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Term != "casino online" {
		t.Fatalf("term mismatch: got %q, want %q", response.Term, "casino online")
	}

	if !response.Changed {
		t.Fatalf("expected changed=true")
	}
}

func TestAdminStopListAddExistingTermReturnsOK(t *testing.T) {
	t.Parallel()

	service := &fakeStopListService{
		addTerm:    "casino",
		addChanged: false,
	}

	mux := newStopListTestMux(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/stop-list",
		bytes.NewBufferString(`{"term":"casino"}`),
	)
	request.Header.Set("Authorization", "Bearer token")

	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestAdminStopListAddRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	mux := newStopListTestMux(&fakeStopListService{})

	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/stop-list",
		bytes.NewBufferString(`{invalid json`),
	)
	request.Header.Set("Authorization", "Bearer token")

	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestAdminStopListAddRejectsEmptyTerm(t *testing.T) {
	t.Parallel()

	service := &fakeStopListService{
		addErr: stoplist.ErrEmptyTerm,
	}

	mux := newStopListTestMux(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/stop-list",
		bytes.NewBufferString(`{"term":"   "}`),
	)
	request.Header.Set("Authorization", "Bearer token")

	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestAdminStopListAddReturnsInternalErrorOnStorageFailure(t *testing.T) {
	t.Parallel()

	service := &fakeStopListService{
		addErr: errors.New("storage failed"),
	}

	mux := newStopListTestMux(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/admin/stop-list",
		bytes.NewBufferString(`{"term":"casino"}`),
	)
	request.Header.Set("Authorization", "Bearer token")

	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestAdminStopListRemove(t *testing.T) {
	t.Parallel()

	service := &fakeStopListService{
		removeTerm:    "casino",
		removeChanged: true,
	}

	mux := newStopListTestMux(service)

	request := httptest.NewRequest(http.MethodDelete, "/admin/stop-list/casino", nil)
	request.Header.Set("Authorization", "Bearer token")

	recorder := httptest.NewRecorder()

	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response stopListTermResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Term != "casino" {
		t.Fatalf("term mismatch: got %q, want %q", response.Term, "casino")
	}

	if !response.Changed {
		t.Fatalf("expected changed=true")
	}
}

func newStopListTestMux(service StopListService) *http.ServeMux {
	mux := http.NewServeMux()

	handler := NewAdminStopListHandler(service, auth.NewTokenAuth("token"))
	handler.Register(mux)

	return mux
}
