package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/kay-kewl/trendstream/internal/snapshot"
)

type SnapshotReader interface {
	Current() *snapshot.Snapshot
}

type TrendsHandler struct {
	snapshots SnapshotReader
}

func NewTrendsHandler(snapshots SnapshotReader) *TrendsHandler {
	return &TrendsHandler{
		snapshots: snapshots,
	}
}

func (h *TrendsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/trends", h.GetTrends)
}

func (h *TrendsHandler) GetTrends(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimit(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	current := h.snapshots.Current()

	if payload, ok := current.PrecomputedJSON(limit); ok {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write(payload)
		return
	}

	response, err := current.Response(limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func parseLimit(r *http.Request) (int, error) {
	rawLimit := r.URL.Query().Get("limit")
	if rawLimit == "" {
		return snapshot.DefaultLimit, nil
	}

	limit, err := strconv.Atoi(rawLimit)
	if err != nil {
		return 0, errors.New("limit must be an integer")
	}

	if err := snapshot.ValidateLimit(limit); err != nil {
		return 0, err
	}

	return limit, nil
}