package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTokenAuthAllowsValidBearerToken(t *testing.T) {
	t.Parallel()

	auth := NewTokenAuth("secret")

	handler := auth.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer secret")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestTokenAuthRejectsMissingHeader(t *testing.T) {
	t.Parallel()

	auth := NewTokenAuth("secret")

	handler := auth.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestTokenAuthRejectsMalformedHeader(t *testing.T) {
	t.Parallel()

	auth := NewTokenAuth("secret")

	handler := auth.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Token secret")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestTokenAuthRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	auth := NewTokenAuth("secret")

	handler := auth.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer wrong")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestTokenAuthRejectsEmptyConfiguredToken(t *testing.T) {
	t.Parallel()

	auth := NewTokenAuth("")

	handler := auth.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer secret")

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status mismatch: got %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}
