package images

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/marcosalvi-01/gowatch/internal/services"
	"github.com/marcosalvi-01/gowatch/logging"

	"github.com/go-chi/chi/v5"
)

var log = logging.Get("images")

type imageService interface {
	GetCachedImage(ctx context.Context, size, imagePath string) (string, error)
	CacheTTL() time.Duration
}

type Handlers struct {
	imageService imageService
}

func NewHandlers(imageService imageService) *Handlers {
	return &Handlers{imageService: imageService}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	log.Debug("registering image routes")
	r.Get("/tmdb/{size}/{imagePath}", h.TMDBImage)
	r.Head("/tmdb/{size}/{imagePath}", h.TMDBImage)
}

func (h *Handlers) TMDBImage(w http.ResponseWriter, r *http.Request) {
	if h.imageService == nil {
		log.Error("tmdb image service not configured")
		http.Error(w, "TMDB image service not configured", http.StatusInternalServerError)
		return
	}

	size := chi.URLParam(r, "size")
	imagePath := chi.URLParam(r, "imagePath")

	cachePath, err := h.imageService.GetCachedImage(r.Context(), size, imagePath)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidTMDBImageSize), errors.Is(err, services.ErrInvalidTMDBImagePath):
			http.Error(w, "invalid TMDB image request", http.StatusBadRequest)
		case errors.Is(err, services.ErrTMDBImageNotFound):
			http.NotFound(w, r)
		case errors.Is(err, services.ErrTMDBImageUnavailable):
			http.Error(w, "failed to fetch TMDB image", http.StatusBadGateway)
		default:
			log.Error("failed to resolve cached TMDB image", "size", size, "imagePath", imagePath, "error", err)
			http.Error(w, "failed to load image", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", cacheMaxAgeSeconds(h.imageService.CacheTTL())))
	http.ServeFile(w, r, cachePath)
}

func cacheMaxAgeSeconds(cacheTTL time.Duration) int64 {
	if cacheTTL <= 0 {
		return 0
	}

	return int64(cacheTTL / time.Second)
}
