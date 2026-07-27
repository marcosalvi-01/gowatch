package htmx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/marcosalvi-01/gowatch/db"
	"github.com/marcosalvi-01/gowatch/internal/common"
	"github.com/marcosalvi-01/gowatch/internal/models"
	"github.com/marcosalvi-01/gowatch/internal/services"
)

func TestHandlers_UpdateListDisplaySort(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "sort@example.com", "Sort User", "hash")
	if err != nil {
		t.Fatal(err)
	}

	listService := services.NewListService(testDB, services.NewMovieService(testDB, nil, time.Hour))
	userCtx := context.WithValue(ctx, common.UserKey, user)
	list, err := listService.CreateList(userCtx, "Sorted", nil, false)
	if err != nil {
		t.Fatal(err)
	}

	h := &Handlers{listService: listService}
	req := httptest.NewRequest(http.MethodPatch, "/lists/1/display-sort", nil)
	req.Form = url.Values{"sort": {string(models.ListMovieSortTitleAsc)}}
	req = req.WithContext(withListRouteParam(userCtx, list.ID))
	res := httptest.NewRecorder()

	h.UpdateListDisplaySort(res, req)

	if res.Header().Get("HX-Trigger") != "refreshListGrid" {
		t.Fatalf("expected grid refresh trigger, got %q", res.Header().Get("HX-Trigger"))
	}
	updated, err := listService.GetListDetails(userCtx, list.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.DisplaySort != models.ListMovieSortTitleAsc {
		t.Fatalf("expected title sort, got %q", updated.DisplaySort)
	}
}

func TestHandlers_UpdateListDetails(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "details@example.com", "Details User", "hash")
	if err != nil {
		t.Fatal(err)
	}
	userCtx := context.WithValue(ctx, common.UserKey, user)
	listService := services.NewListService(testDB, services.NewMovieService(testDB, nil, time.Hour))
	list, err := listService.CreateList(userCtx, "Original", nil, false)
	if err != nil {
		t.Fatal(err)
	}

	h := &Handlers{listService: listService}
	req := httptest.NewRequest(http.MethodPatch, "/lists/1", nil)
	req.Form = url.Values{"title": {"Updated"}, "description": {"Description"}}
	req = req.WithContext(withListRouteParam(userCtx, list.ID))
	res := httptest.NewRecorder()

	h.UpdateListDetails(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
	if res.Header().Get("HX-Redirect") != "/list/1" {
		t.Fatalf("expected redirect to list, got %q", res.Header().Get("HX-Redirect"))
	}
	updated, err := listService.GetListDetails(userCtx, list.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Updated" || updated.Description == nil || *updated.Description != "Description" {
		t.Fatalf("expected updated details, got name=%q description=%v", updated.Name, updated.Description)
	}
}

func withListRouteParam(ctx context.Context, listID int64) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(listID, 10))

	return context.WithValue(ctx, chi.RouteCtxKey, rctx)
}
