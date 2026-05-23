package ingest

import (
	"net/http"

	"github.com/kay-kewl/trendstream/internal/contract"
)

type HTTPProcessor struct {
	processor *Processor
}

func NewHTTPProcessor(processor *Processor) *HTTPProcessor {
	return &HTTPProcessor{
		processor: processor,
	}
}

func (p *HTTPProcessor) ProcessHTTP(r *http.Request, event contract.SearchEvent) Result {
	return p.processor.Process(r.Context(), event)
}
