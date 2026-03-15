package pages

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/marcosalvi-01/gowatch/internal/services"
)

func TestHandlers_PersonPage_InvalidIDReturnsBadRequest(t *testing.T) {
	h := &Handlers{}

	req := httptest.NewRequest(http.MethodGet, "/person/not-a-number", nil)
	req = req.WithContext(withRouteParam(req.Context(), "id", "not-a-number"))
	res := httptest.NewRecorder()

	h.PersonPage(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}

func TestHandlers_PersonPage_ServiceFailureReturnsInternalServerError(t *testing.T) {
	h := &Handlers{
		tmdbService: services.NewMovieService(nil, nil, time.Hour),
	}

	req := httptest.NewRequest(http.MethodGet, "/person/42", nil)
	req = req.WithContext(withRouteParam(req.Context(), "id", "42"))
	res := httptest.NewRecorder()

	h.PersonPage(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, res.Code)
	}
}

func withRouteParam(ctx context.Context, key, value string) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)

	return context.WithValue(ctx, chi.RouteCtxKey, rctx)
}
