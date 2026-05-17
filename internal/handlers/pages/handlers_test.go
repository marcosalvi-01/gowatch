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

func TestHandlers_SearchPage_EmptyQueryReturnsOK(t *testing.T) {
	h := &Handlers{}

	req := httptest.NewRequest(http.MethodGet, "/search?q=", nil)
	res := httptest.NewRecorder()

	h.SearchPage(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
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

func TestHandlers_ListPage_CustomEditModeToggle(t *testing.T) {
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

	movie := &models.MovieDetails{Movie: models.Movie{ID: 1, Title: "Alien"}}
	if err := testDB.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}

	listService := services.NewListService(testDB, services.NewMovieService(testDB, nil, time.Hour))
	list, err := listService.CreateList(context.WithValue(ctx, common.UserKey, user), "Sci-Fi", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := listService.AddMovieToList(context.WithValue(ctx, common.UserKey, user), list.ID, movie.Movie.ID, nil); err != nil {
		t.Fatal(err)
	}

	h := &Handlers{listService: listService}

	t.Run("view mode by default", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/list/1?sort=custom", nil)
		req = req.WithContext(withRouteParam(context.WithValue(req.Context(), common.UserKey, user), "id", strconv.FormatInt(list.ID, 10)))
		res := httptest.NewRecorder()

		h.ListPage(res, req)

		body := res.Body.String()
		if !strings.Contains(body, "Edit order") {
			t.Fatal("expected Edit order button in default custom view mode")
		}
		if strings.Contains(body, "Done") {
			t.Fatal("did not expect Done button in default custom view mode")
		}
	})

	t.Run("done button in edit mode", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/list/1?sort=custom&edit=1", nil)
		req = req.WithContext(withRouteParam(context.WithValue(req.Context(), common.UserKey, user), "id", strconv.FormatInt(list.ID, 10)))
		res := httptest.NewRecorder()

		h.ListPage(res, req)

		body := res.Body.String()
		if !strings.Contains(body, "Done") {
			t.Fatal("expected Done button in custom edit mode")
		}
		if !strings.Contains(body, "/htmx/lists/1/movie-grid?sort=custom&amp;edit=1") {
			t.Fatal("expected grid URL to preserve custom edit mode")
		}
	})

	t.Run("non custom sort ignores edit flag", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/list/1?sort=title_asc&edit=1", nil)
		req = req.WithContext(withRouteParam(context.WithValue(req.Context(), common.UserKey, user), "id", strconv.FormatInt(list.ID, 10)))
		res := httptest.NewRecorder()

		h.ListPage(res, req)

		body := res.Body.String()
		if strings.Contains(body, "Done") {
			t.Fatal("did not expect Done button for non-custom sort")
		}
		if strings.Contains(body, "Reorder movies with arrow controls.") {
			t.Fatal("did not expect reorder helper text for non-custom sort")
		}
	})
}

func withRouteParam(ctx context.Context, key, value string) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)

	return context.WithValue(ctx, chi.RouteCtxKey, rctx)
}
