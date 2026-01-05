// Package middleware provides HTTP middleware functions for logging, error recovery, and content type handling.
package middleware

import (
	"net/http"
	"strings"

	"gowatch/logging"
)

var log = logging.Get("http")

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)

		if !strings.HasPrefix(r.URL.Path, "/static/") {
			log.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"status", rw.status,
			)
		}
	})
}

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic recovered", "error", err, "path", r.URL.Path)
				if strings.HasPrefix(r.URL.Path, "/api") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					if _, err := w.Write([]byte(`{"error": "Internal Server Error"}`)); err != nil {
						log.Error("failed to write error response", "error", err)
					}
				} else {
					w.Header().Set("Content-Type", "text/html")
					w.WriteHeader(http.StatusInternalServerError)
					if _, err := w.Write([]byte(`<!DOCTYPE html><html><head><title>500 Internal Server Error</title></head><body><h1>500 Internal Server Error</h1><p>Something went wrong. <a href="/home">Go Home</a></p></body></html>`)); err != nil {
						log.Error("failed to write error response", "error", err)
					}
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}
