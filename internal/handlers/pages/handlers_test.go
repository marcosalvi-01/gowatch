package pages

import (
	"bytes"
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
	uipages "github.com/marcosalvi-01/gowatch/internal/ui/pages"
)

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
	h := &Handlers{
		tmdbService: services.NewMovieService(nil, nil, time.Hour),
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

func TestMovieListAction(t *testing.T) {
	testCases := []struct {
		name       string
		hasLists   bool
		expected   []string
		unexpected string
	}{
		{
			name:     "no lists opens create list dialog",
			hasLists: false,
			expected: []string{
				`data-tui-dialog-target="add-to-list-dialog"`,
				`id="add-movie-to-list-dialog"`,
			},
		},
		{
			name:       "lists opens add movie dialog",
			hasLists:   true,
			expected:   []string{`id="add-movie-to-list-dialog"`},
			unexpected: `data-tui-dialog-target="add-to-list-dialog"`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var body bytes.Buffer
			err := uipages.Movie(
				models.MovieDetails{Movie: models.Movie{ID: 1, Title: "Alien"}},
				models.WatchedMovieRecords{},
				false,
				testCase.hasLists,
			).Render(context.Background(), &body)
			if err != nil {
				t.Fatal(err)
			}

			html := body.String()
			for _, expected := range testCase.expected {
				if !strings.Contains(html, expected) {
					t.Fatalf("expected rendered movie page to contain %q", expected)
				}
			}
			if testCase.unexpected != "" && strings.Contains(html, testCase.unexpected) {
				t.Fatalf("did not expect rendered movie page to contain %q", testCase.unexpected)
			}
		})
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
	userCtx := context.WithValue(ctx, common.UserKey, user)
	list, err := listService.CreateList(userCtx, "Sci-Fi", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := listService.AddMovieToList(userCtx, list.ID, movie.Movie.ID, nil); err != nil {
		t.Fatal(err)
	}
	watchlist, err := listService.CreateList(userCtx, "Watchlist", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := listService.AddMovieToList(userCtx, watchlist.ID, movie.Movie.ID, nil); err != nil {
		t.Fatal(err)
	}

	h := &Handlers{listService: listService}

	t.Run("view mode by default", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/list/1", nil)
		req = req.WithContext(withRouteParam(context.WithValue(req.Context(), common.UserKey, user), strconv.FormatInt(list.ID, 10)))
		res := httptest.NewRecorder()

		h.ListPage(res, req)

		body := res.Body.String()
		if !strings.Contains(body, "Edit list") {
			t.Fatal("expected Edit list button in default custom view mode")
		}
		if strings.Contains(body, "Delete List") {
			t.Fatal("did not expect Delete List button in view mode")
		}
		if strings.Contains(body, "Done") {
			t.Fatal("did not expect Done button in default custom view mode")
		}
	})

	t.Run("done button in edit mode", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/list/1?edit=1", nil)
		req = req.WithContext(withRouteParam(context.WithValue(req.Context(), common.UserKey, user), strconv.FormatInt(list.ID, 10)))
		res := httptest.NewRecorder()

		h.ListPage(res, req)

		body := res.Body.String()
		if !strings.Contains(body, "Done") {
			t.Fatal("expected Done button in custom edit mode")
		}
		if !strings.Contains(body, "/htmx/lists/1/movie-grid") {
			t.Fatal("expected grid URL")
		}
		if !strings.Contains(body, "Delete List") {
			t.Fatal("expected Delete List button in edit mode")
		}
		if !strings.Contains(body, `name="title"`) || !strings.Contains(body, `name="description"`) {
			t.Fatal("expected list details fields in edit mode")
		}
		if strings.Contains(body, "Save changes") {
			t.Fatal("did not expect separate save button")
		}
	})

	t.Run("non custom sort starts in view mode", func(t *testing.T) {
		if err := listService.SetListDisplaySort(userCtx, list.ID, models.ListMovieSortTitleAsc); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, "/list/1", nil)
		//nolint:contextcheck // withRouteParam derives context from this request.
		req = req.WithContext(withRouteParam(context.WithValue(req.Context(), common.UserKey, user), strconv.FormatInt(list.ID, 10)))
		res := httptest.NewRecorder()

		h.ListPage(res, req)

		body := res.Body.String()
		if !strings.Contains(body, "Edit list") {
			t.Fatal("expected Edit list button for non-custom sort")
		}
		if strings.Contains(body, "Done") {
			t.Fatal("did not expect Done button in view mode")
		}
		if !strings.Contains(body, "Display order") {
			t.Fatal("expected display order control in view mode")
		}
	})

	t.Run("non custom sort can be edited", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/list/1?edit=1", nil)
		req = req.WithContext(withRouteParam(context.WithValue(req.Context(), common.UserKey, user), strconv.FormatInt(list.ID, 10)))
		res := httptest.NewRecorder()

		h.ListPage(res, req)

		body := res.Body.String()
		if !strings.Contains(body, "Done") {
			t.Fatal("expected Done button in edit mode")
		}
		if !strings.Contains(body, "/htmx/lists/1/movie-grid") {
			t.Fatal("expected grid URL")
		}
		if strings.Contains(body, "Reorder movies with arrow controls.") {
			t.Fatal("did not expect reorder helper text for non-custom sort")
		}
	})

	t.Run("watchlist uses shared page and canonical route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/list/2", nil)
		req = req.WithContext(withRouteParam(context.WithValue(req.Context(), common.UserKey, user), strconv.FormatInt(watchlist.ID, 10)))
		res := httptest.NewRecorder()

		h.ListPage(res, req)

		if res.Code != http.StatusFound {
			t.Fatalf("expected redirect status %d, got %d", http.StatusFound, res.Code)
		}
		if location := res.Header().Get("Location"); location != "/watchlist" {
			t.Fatalf("expected watchlist redirect, got %q", location)
		}

		req = httptest.NewRequest(http.MethodGet, "/watchlist?edit=1", nil)
		req = req.WithContext(context.WithValue(req.Context(), common.UserKey, user))
		res = httptest.NewRecorder()

		h.Watchlist(res, req)

		body := res.Body.String()
		for _, expected := range []string{"My Watchlist", "Upcoming / Released", "Done"} {
			if !strings.Contains(body, expected) {
				t.Fatalf("expected watchlist page to contain %q", expected)
			}
		}
		if !strings.Contains(body, "Custom order") {
			t.Fatal("expected custom order for watchlist")
		}
		if strings.Contains(body, "Save changes") {
			t.Fatal("did not expect list details form for watchlist")
		}
	})
}

func withRouteParam(ctx context.Context, value string) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", value)

	return context.WithValue(ctx, chi.RouteCtxKey, rctx)
}
