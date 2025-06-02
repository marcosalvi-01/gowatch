package prometheus

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		timer := prometheus.NewTimer(requestDuration.WithLabelValues(r.URL.Path, r.Method))
		defer timer.ObserveDuration()

		next.ServeHTTP(rec, r)

		statusCode := strconv.Itoa(rec.status)
		requestCounter.WithLabelValues(r.URL.Path, r.Method, statusCode).Inc()

		if rec.status >= 500 {
			errorCounter.Inc()
		}
	})
}
