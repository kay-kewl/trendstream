package stoplist

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

type Store interface {
	Load() ([]string, error)
	Save(terms []string) error
}

type FileStore struct {
	path string
}

type filePayload struct {
	Terms []string `json:"terms"`
}

func NewFileStore(path string) *FileStore {
	return &FileStore{
		path: path,
	}
}

func (s *FileStore) Load() ([]string, error) {
	payload, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}

		return nil, err
	}

	var data filePayload
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}

	normalized := make([]string, 0, len(data.Terms))
	seen := make(map[string]struct{}, len(data.Terms))

	for _, rawTerm := range data.Terms {
		term, err := NormalizeTerm(rawTerm)
		if err != nil {
			continue
		}

		if _, exists := seen[term]; exists {
			continue
		}

		seen[term] = struct{}{}
		normalized = append(normalized, term)
	}

	sort.Strings(normalized)

	return normalized, nil
}

func (s *FileStore) Save(terms []string) error {
	normalized := make([]string, 0, len(terms))
	seen := make(map[string]struct{}, len(terms))

	for _, rawTerm := range terms {
		term, err := NormalizeTerm(rawTerm)
		if err != nil {
			continue
		}

		if _, exists := seen[term]; exists {
			continue
		}

		seen[term] = struct{}{}
		normalized = append(normalized, term)
	}

	sort.Strings(normalized)

	data := filePayload{
		Terms: normalized,
	}

	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	payload = append(payload, '\n')

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(dir, ".stoplist-*.tmp")
	if err != nil {
		return err
	}

	tempPath := tempFile.Name()

	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := tempFile.Write(payload); err != nil {
		_ = tempFile.Close()
		return err
	}

	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return err
	}

	if err := tempFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tempPath, s.path); err != nil {
		return err
	}

	cleanup = false

	return nil
}
