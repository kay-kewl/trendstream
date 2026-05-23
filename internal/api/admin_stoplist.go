package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kay-kewl/trendstream/internal/auth"
	"github.com/kay-kewl/trendstream/internal/stoplist"
)

type StopListService interface {
	Terms() []string
	Add(rawTerm string) (string, bool, error)
	Remove(rawTerm string) (string, bool, error)
}

type AdminStopListHandler struct {
	service StopListService
	auth    *auth.TokenAuth
}

type stopListResponse struct {
	Terms []string `json:"terms"`
}

type stopListTermRequest struct {
	Term string `json:"term"`
}

type stopListTermResponse struct {
	Term    string `json:"term"`
	Changed bool   `json:"changed"`
}

func NewAdminStopListHandler(service StopListService, tokenAuth *auth.TokenAuth) *AdminStopListHandler {
	return &AdminStopListHandler{
		service: service,
		auth:    tokenAuth,
	}
}

func (h *AdminStopListHandler) Register(mux *http.ServeMux) {
	mux.Handle("GET /admin/stop-list", h.auth.Wrap(http.HandlerFunc(h.List)))
	mux.Handle("POST /admin/stop-list", h.auth.Wrap(http.HandlerFunc(h.Add)))
	mux.Handle("DELETE /admin/stop-list/{term}", h.auth.Wrap(http.HandlerFunc(h.Remove)))
}

func (h *AdminStopListHandler) List(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, stopListResponse{
		Terms: h.service.Terms(),
	})
}

func (h *AdminStopListHandler) Add(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var request stopListTermRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	term, changed, err := h.service.Add(request.Term)
	if err != nil {
		writeStopListError(w, err)
		return
	}

	statusCode := http.StatusOK
	if changed {
		statusCode = http.StatusCreated
	}

	writeJSON(w, statusCode, stopListTermResponse{
		Term:    term,
		Changed: changed,
	})
}

func (h *AdminStopListHandler) Remove(w http.ResponseWriter, r *http.Request) {
	rawTerm := r.PathValue("term")

	term, changed, err := h.service.Remove(rawTerm)
	if err != nil {
		writeStopListError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, stopListTermResponse{
		Term:    term,
		Changed: changed,
	})
}

func writeStopListError(w http.ResponseWriter, err error) {
	if errors.Is(err, stoplist.ErrEmptyTerm) {
		writeError(w, http.StatusBadRequest, "term is required")
		return
	}

	writeError(w, http.StatusInternalServerError, "failed to update stop-list")
}
