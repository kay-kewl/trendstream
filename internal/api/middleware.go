package api

import (
	"net/http"
	"strings"
	"time"
)

type HTTPMetrics interface {
	ObserveHTTPRequest(server string, method string, path string, statusCode int, duration time.Duration)
}

func MetricsMiddleware(serverName string, observer HTTPMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if observer == nil {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			recorder := &statusRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(recorder, r)

			observer.ObserveHTTPRequest(
				serverName,
				r.Method,
				routePattern(r),
				recorder.statusCode,
				time.Since(startedAt),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func routePattern(r *http.Request) string {
	path := r.URL.Path

	switch {
	case path == "/healthz":
		return "/healthz"
	case path == "/readyz":
		return "/readyz"
	case path == "/v1/trends":
		return "/v1/trends"
	case path == "/admin/events":
		return "/admin/events"
	case path == "/admin/stop-list":
		return "/admin/stop-list"
	case strings.HasPrefix(path, "/admin/stop-list/"):
		return "/admin/stop-list/{term}"
	case path == "/metrics":
		return "/metrics"
	case strings.HasPrefix(path, "/debug/pprof"):
		return "/debug/pprof/*"
	default:
		return "unknown"
	}
}
