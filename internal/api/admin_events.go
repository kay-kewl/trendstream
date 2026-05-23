package api

import (
	"encoding/json"
	"net/http"

	"github.com/burtonjake686/trendstream/internal/auth"
	"github.com/burtonjake686/trendstream/internal/contract"
	"github.com/burtonjake686/trendstream/internal/ingest"
)

const maxAdminEventBodyBytes = 64 * 1024

type AdminEventProcessor interface {
	ProcessHTTP(r *http.Request, event contract.SearchEvent) ingest.Result
}

type AdminEventsHandler struct {
	processor AdminEventProcessor
	auth      *auth.TokenAuth
}

func NewAdminEventsHandler(processor AdminEventProcessor, tokenAuth *auth.TokenAuth) *AdminEventsHandler {
	return &AdminEventsHandler{
		processor: processor,
		auth:      tokenAuth,
	}
}

func (h *AdminEventsHandler) Register(mux *http.ServeMux) {
	mux.Handle("POST /admin/events", h.auth.Wrap(http.HandlerFunc(h.AddEvent)))
}

func (h *AdminEventsHandler) AddEvent(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	r.Body = http.MaxBytesReader(w, r.Body, maxAdminEventBodyBytes)

	var event contract.SearchEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	result := h.processor.ProcessHTTP(r, event)

	statusCode := http.StatusAccepted
	if !result.Accepted {
		statusCode = http.StatusOK
	}

	if result.Reason == ingest.ReasonInvalidEvent || result.Reason == ingest.ReasonEmptyQuery {
		statusCode = http.StatusBadRequest
	}

	writeJSON(w, statusCode, result)
}