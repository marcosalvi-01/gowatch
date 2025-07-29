// Package middleware contains HTTP middleware functions that are applied
// to request processing pipeline. This includes logging, authentication,
// content-type setting, recovery, and other cross-cutting concerns.
package middleware

import "net/http"

func JSONMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
