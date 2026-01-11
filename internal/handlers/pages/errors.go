package pages

import (
	"net/http"

	"github.com/marcosalvi-01/gowatch/internal/ui/pages"

	"github.com/a-h/templ"
)

func (h *Handlers) Error404Page(w http.ResponseWriter, r *http.Request) {
	log.Debug("serving 404 error page")

	templ.Handler(pages.Error404()).ServeHTTP(w, r)

	log.Info("404 error page served")
}

func render500Error(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	templ.Handler(pages.Error500()).ServeHTTP(w, r)
}
