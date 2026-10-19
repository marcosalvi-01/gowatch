package pages

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/marcosalvi-01/gowatch/db"
	"github.com/marcosalvi-01/gowatch/internal/common"
	"github.com/marcosalvi-01/gowatch/internal/models"
	"github.com/marcosalvi-01/gowatch/internal/services"
)

func newTestSeerrService(t *testing.T) *services.SeerrService {
	t.Helper()
	service, err := services.NewSeerrService(models.SeerrIntegration{}, nil)
	if err != nil {
		t.Fatalf("NewSeerrService() error = %v", err)
	}
	return service
}

func TestNewHandlersRequiresDependencies(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewHandlers() did not panic without required dependencies")
		}
	}()

	NewHandlers(nil, nil, nil, nil, nil, nil)
}

func TestHandlers_PersonPage_InvalidIDReturnsBadRequest(t *testing.T) {
	h := &Handlers{}

	req := httptest.NewRequest(http.MethodGet, "/person/not-a-number", nil)
	req = req.WithContext(withRouteParam(req.Context(), "not-a-number"))
	res := httptest.NewRecorder()

	h.PersonPage(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}

func TestHandlers_PersonPage_ServiceFailureReturnsInternalServerError(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	h := &Handlers{
		movieService: services.NewMovieService(testDB, newTestSeerrService(t), time.Hour),
	}

	req := httptest.NewRequest(http.MethodGet, "/person/42", nil)
	req = req.WithContext(withRouteParam(req.Context(), "42"))
	res := httptest.NewRecorder()

	h.PersonPage(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, res.Code)
	}
}

func TestHandlers_SearchPage_EmptyQueryReturnsOK(t *testing.T) {
	h := &Handlers{}

	req := httptest.NewRequest(http.MethodGet, "/search?q=", nil)
	res := httptest.NewRecorder()

	h.SearchPage(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
}

func TestHandlers_SearchPage_EmptyQueryFragment(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/search?q=", nil)
	req.Header.Set(htmxRequestHeader, htmxRequestHeaderValue)
	res := httptest.NewRecorder()

	(&Handlers{}).SearchPage(res, req)

	if res.Code != http.StatusOK || res.Header().Get("HX-Trigger") != "refreshSidebar" ||
		!strings.Contains(res.Body.String(), "No search results") || strings.Contains(res.Body.String(), "<!DOCTYPE html>") {
		t.Fatalf("fragment status = %d, trigger = %q, body = %q", res.Code, res.Header().Get("HX-Trigger"), res.Body.String())
	}
}

func TestHandlers_SearchPage_WhitespaceQueryReturnsOK(t *testing.T) {
	h := &Handlers{}

	req := httptest.NewRequest(http.MethodGet, "/search?q=%20%20%20", nil)
	res := httptest.NewRecorder()

	h.SearchPage(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
}

func TestHandlers_SearchPage_QueryTooLongReturnsBadRequest(t *testing.T) {
	h := &Handlers{}
	query := strings.Repeat("a", maxSearchQueryLength+1)

	req := httptest.NewRequest(http.MethodGet, "/search?q="+query, nil)
	res := httptest.NewRecorder()

	h.SearchPage(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}

func TestHandlers_ListPageRedirectsWatchlist(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "list@example.com", "List User", "hash")
	if err != nil {
		t.Fatal(err)
	}
	userCtx := context.WithValue(ctx, common.UserKey, user)
	listService := services.NewListService(testDB, services.NewMovieService(testDB, newTestSeerrService(t), time.Hour))
	watchlist, err := listService.CreateList(userCtx, "Watchlist", nil, true)
	if err != nil {
		t.Fatal(err)
	}

	h := &Handlers{listService: listService}
	listID := strconv.FormatInt(watchlist.ID, 10)
	req := httptest.NewRequest(http.MethodGet, "/list/"+listID, nil)
	req = req.WithContext(withRouteParam(userCtx, listID))
	res := httptest.NewRecorder()

	h.ListPage(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusFound)
	}
	if location := res.Header().Get("Location"); location != "/watchlist" {
		t.Fatalf("Location = %q, want %q", location, "/watchlist")
	}
}

func withRouteParam(ctx context.Context, value string) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", value)

	return context.WithValue(ctx, chi.RouteCtxKey, rctx)
}
