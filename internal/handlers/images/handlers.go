package images

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
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
	if err := serveCachedImageFile(w, r, cachePath); err != nil {
		log.Error("failed to serve cached TMDB image", "cachePath", cachePath, "error", err)
		http.Error(w, "failed to load image", http.StatusInternalServerError)
		return
	}
}

func serveCachedImageFile(w http.ResponseWriter, r *http.Request, cachePath string) error {
	// #nosec G304,G703 -- path comes from image service cache resolver with prior validation.
	file, err := os.Open(cachePath)
	if err != nil {
		return fmt.Errorf("open cached image: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat cached image: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("cached image path is directory")
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	return nil
}

func cacheMaxAgeSeconds(cacheTTL time.Duration) int64 {
	if cacheTTL <= 0 {
		return 0
	}

	return int64(cacheTTL / time.Second)
}
