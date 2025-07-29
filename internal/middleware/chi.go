package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// TODO: implement these instead of using chi middleware

func Logger(next http.Handler) http.Handler {
	return middleware.Logger(next)
}

func Recoverer(next http.Handler) http.Handler {
	return middleware.Recoverer(next)
}
