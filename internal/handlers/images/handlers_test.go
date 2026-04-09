package images

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/marcosalvi-01/gowatch/internal/services"

	"github.com/go-chi/chi/v5"
)

type stubImageService struct {
	cachePath string
	err       error
	cacheTTL  time.Duration
}

func (s stubImageService) GetCachedImage(_ context.Context, _, _ string) (string, error) {
	return s.cachePath, s.err
}

func (s stubImageService) CacheTTL() time.Duration {
	return s.cacheTTL
}

func TestHandlers_TMDBImage_ServesCachedFile(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), "poster.jpg")
	if err := os.WriteFile(cacheFile, []byte("cached image"), 0o600); err != nil {
		t.Fatalf("failed to write cached image: %v", err)
	}

	router := chi.NewRouter()
	NewHandlers(stubImageService{cachePath: cacheFile, cacheTTL: time.Hour}).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/tmdb/w500/poster.jpg", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
	if body := res.Body.String(); body != "cached image" {
		t.Fatalf("expected cached file body %q, got %q", "cached image", body)
	}
	if cacheControl := res.Header().Get("Cache-Control"); cacheControl != "public, max-age=3600" {
		t.Fatalf("expected cache header %q, got %q", "public, max-age=3600", cacheControl)
	}
}

func TestHandlers_TMDBImage_InvalidSizeReturnsBadRequest(t *testing.T) {
	router := chi.NewRouter()
	NewHandlers(stubImageService{err: services.ErrInvalidTMDBImageSize}).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/tmdb/w999/poster.jpg", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}

func TestHandlers_TMDBImage_InvalidPathReturnsBadRequest(t *testing.T) {
	router := chi.NewRouter()
	NewHandlers(stubImageService{err: services.ErrInvalidTMDBImagePath}).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/tmdb/w500/bad-path.jpg", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}

func TestHandlers_TMDBImage_NotFoundReturns404(t *testing.T) {
	router := chi.NewRouter()
	NewHandlers(stubImageService{err: services.ErrTMDBImageNotFound}).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/tmdb/w500/missing.jpg", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, res.Code)
	}
}

func TestHandlers_TMDBImage_UpstreamFailureReturnsBadGateway(t *testing.T) {
	router := chi.NewRouter()
	NewHandlers(stubImageService{err: services.ErrTMDBImageUnavailable}).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/tmdb/w500/poster.jpg", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, res.Code)
	}
}

func TestHandlers_TMDBImage_UnexpectedErrorReturnsInternalServerError(t *testing.T) {
	router := chi.NewRouter()
	NewHandlers(stubImageService{err: errors.New("boom")}).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/tmdb/w500/poster.jpg", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, res.Code)
	}
}
