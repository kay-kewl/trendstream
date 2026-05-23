package stoplist

import (
	"errors"
	"testing"
)

type memoryStore struct {
	terms   []string
	loadErr error
	saveErr error
}

func (s *memoryStore) Load() ([]string, error) {
	if s.loadErr != nil {
		return nil, s.loadErr
	}

	return append([]string(nil), s.terms...), nil
}

func (s *memoryStore) Save(terms []string) error {
	if s.saveErr != nil {
		return s.saveErr
	}

	s.terms = append([]string(nil), terms...)

	return nil
}

func TestNewServiceLoadsInitialTerms(t *testing.T) {
	t.Parallel()

	store := &memoryStore{
		terms: []string{"casino"},
	}

	service, err := NewService(store)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	if !service.Contains("casino") {
		t.Fatalf("expected initial term to be loaded")
	}
}

func TestNewServiceReturnsLoadError(t *testing.T) {
	t.Parallel()

	loadErr := errors.New("load failed")

	_, err := NewService(&memoryStore{
		loadErr: loadErr,
	})

	if !errors.Is(err, loadErr) {
		t.Fatalf("expected load error, got %v", err)
	}
}

func TestServiceAdd(t *testing.T) {
	t.Parallel()

	store := &memoryStore{}
	service, err := NewService(store)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	term, changed, err := service.Add(" Casino   Online ")
	if err != nil {
		t.Fatalf("failed to add term: %v", err)
	}

	if term != "casino online" {
		t.Fatalf("term mismatch: got %q, want %q", term, "casino online")
	}

	if !changed {
		t.Fatalf("expected changed=true")
	}

	if !service.Contains("casino online") {
		t.Fatalf("expected service to contain added term")
	}

	if len(store.terms) != 1 || store.terms[0] != "casino online" {
		t.Fatalf("unexpected persisted terms: %#v", store.terms)
	}
}

func TestServiceAddExistingTermIsNoop(t *testing.T) {
	t.Parallel()

	store := &memoryStore{
		terms: []string{"casino"},
	}

	service, err := NewService(store)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	term, changed, err := service.Add("CASINO")
	if err != nil {
		t.Fatalf("failed to add term: %v", err)
	}

	if term != "casino" {
		t.Fatalf("term mismatch: got %q, want %q", term, "casino")
	}

	if changed {
		t.Fatalf("expected changed=false")
	}

	if len(store.terms) != 1 {
		t.Fatalf("store should not be changed, got %#v", store.terms)
	}
}

func TestServiceRemove(t *testing.T) {
	t.Parallel()

	store := &memoryStore{
		terms: []string{"casino", "spam"},
	}

	service, err := NewService(store)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	term, changed, err := service.Remove("CASINO")
	if err != nil {
		t.Fatalf("failed to remove term: %v", err)
	}

	if term != "casino" {
		t.Fatalf("term mismatch: got %q, want %q", term, "casino")
	}

	if !changed {
		t.Fatalf("expected changed=true")
	}

	if service.Contains("casino") {
		t.Fatalf("removed term should not exist")
	}

	if !service.Contains("spam") {
		t.Fatalf("unrelated term should remain")
	}
}

func TestServiceRemoveMissingTermIsNoop(t *testing.T) {
	t.Parallel()

	store := &memoryStore{
		terms: []string{"casino"},
	}

	service, err := NewService(store)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	term, changed, err := service.Remove("missing")
	if err != nil {
		t.Fatalf("failed to remove term: %v", err)
	}

	if term != "missing" {
		t.Fatalf("term mismatch: got %q, want %q", term, "missing")
	}

	if changed {
		t.Fatalf("expected changed=false")
	}
}

func TestServiceDoesNotPublishWhenSaveFails(t *testing.T) {
	t.Parallel()

	saveErr := errors.New("save failed")

	store := &memoryStore{
		saveErr: saveErr,
	}

	service, err := NewService(store)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	_, _, err = service.Add("casino")
	if !errors.Is(err, saveErr) {
		t.Fatalf("expected save error, got %v", err)
	}

	if service.Contains("casino") {
		t.Fatalf("term should not be published if save failed")
	}
}
