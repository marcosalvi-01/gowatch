package pages

import (
	"gowatch/internal/ui/pages"
	"net/http"

	"github.com/a-h/templ"
)

func (h *Handlers) Error404Page(w http.ResponseWriter, r *http.Request) {
	log.Debug("serving 404 error page")

	templ.Handler(pages.Error404()).ServeHTTP(w, r)

	log.Info("404 error page served")
}

func (h *Handlers) Error500Page(w http.ResponseWriter, r *http.Request) {
	log.Debug("serving 500 error page")

	render500Error(w, r)

	log.Info("500 error page served")
}

func render500Error(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	templ.Handler(pages.Error500()).ServeHTTP(w, r)
}
