package middleware

import "net/http"

func HTMLMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/html")
		next.ServeHTTP(w, r)
	})
}
