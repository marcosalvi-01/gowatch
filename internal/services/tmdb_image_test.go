package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTMDBImageService_GetCachedImage_CacheHit(t *testing.T) {
	cacheDir := t.TempDir()
	service := newTestTMDBImageService(cacheDir, nil, "http://example.com")

	cachePath := filepath.Join(cacheDir, "w500", "poster.jpg")
	writeCachedTMDBImage(t, cachePath, []byte("cached image"), time.Now())

	resolvedPath, err := service.GetCachedImage(context.Background(), "w500", "poster.jpg")
	if err != nil {
		t.Fatalf("expected cached image lookup to succeed, got error: %v", err)
	}

	if resolvedPath != cachePath {
		t.Fatalf("expected cache path %q, got %q", cachePath, resolvedPath)
	}
}

func TestTMDBImageService_GetCachedImage_DownloadsOnCacheMiss(t *testing.T) {
	cacheDir := t.TempDir()
	var requestCount atomic.Int32
	requestPath := make(chan string, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		requestPath <- r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fresh image"))
	}))
	defer server.Close()

	service := newTestTMDBImageService(cacheDir, server.Client(), server.URL)

	resolvedPath, err := service.GetCachedImage(context.Background(), "w500", "poster.jpg")
	if err != nil {
		t.Fatalf("expected image fetch to succeed, got error: %v", err)
	}

	assertTMDBImageContent(t, resolvedPath, []byte("fresh image"))
	if requestCount.Load() != 1 {
		t.Fatalf("expected 1 upstream request, got %d", requestCount.Load())
	}
	if gotPath := <-requestPath; gotPath != "/w500/poster.jpg" {
		t.Fatalf("expected request path %q, got %q", "/w500/poster.jpg", gotPath)
	}
}

func TestTMDBImageService_GetCachedImage_RefreshesExpiredCache(t *testing.T) {
	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, "w500", "poster.jpg")
	writeCachedTMDBImage(t, cachePath, []byte("stale image"), time.Now().Add(-2*time.Hour))

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("refreshed image"))
	}))
	defer server.Close()

	service := newTestTMDBImageService(cacheDir, server.Client(), server.URL)

	resolvedPath, err := service.GetCachedImage(context.Background(), "w500", "poster.jpg")
	if err != nil {
		t.Fatalf("expected expired cache refresh to succeed, got error: %v", err)
	}

	assertTMDBImageContent(t, resolvedPath, []byte("refreshed image"))
	if requestCount.Load() != 1 {
		t.Fatalf("expected 1 upstream refresh request, got %d", requestCount.Load())
	}
}

func TestTMDBImageService_GetCachedImage_ServesStaleCacheOnUpstreamFailure(t *testing.T) {
	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, "w500", "poster.jpg")
	writeCachedTMDBImage(t, cachePath, []byte("stale image"), time.Now().Add(-2*time.Hour))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	service := newTestTMDBImageService(cacheDir, server.Client(), server.URL)

	resolvedPath, err := service.GetCachedImage(context.Background(), "w500", "poster.jpg")
	if err != nil {
		t.Fatalf("expected stale cache fallback to succeed, got error: %v", err)
	}

	if resolvedPath != cachePath {
		t.Fatalf("expected stale cache path %q, got %q", cachePath, resolvedPath)
	}
	assertTMDBImageContent(t, resolvedPath, []byte("stale image"))
}

func TestTMDBImageService_GetCachedImage_InvalidSize(t *testing.T) {
	service := newTestTMDBImageService(t.TempDir(), nil, "http://example.com")

	_, err := service.GetCachedImage(context.Background(), "w999", "poster.jpg")
	if !errors.Is(err, ErrInvalidTMDBImageSize) {
		t.Fatalf("expected invalid size error, got %v", err)
	}
}

func TestTMDBImageService_GetCachedImage_InvalidPath(t *testing.T) {
	service := newTestTMDBImageService(t.TempDir(), nil, "http://example.com")

	_, err := service.GetCachedImage(context.Background(), "w500", "../../poster.jpg")
	if !errors.Is(err, ErrInvalidTMDBImagePath) {
		t.Fatalf("expected invalid path error, got %v", err)
	}
}

func TestTMDBImageService_GetCachedImage_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	service := newTestTMDBImageService(t.TempDir(), server.Client(), server.URL)

	_, err := service.GetCachedImage(context.Background(), "w500", "poster.jpg")
	if !errors.Is(err, ErrTMDBImageNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestTMDBImageService_GetCachedImage_CollapsesConcurrentMisses(t *testing.T) {
	cacheDir := t.TempDir()
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		time.Sleep(25 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("shared image"))
	}))
	defer server.Close()

	service := newTestTMDBImageService(cacheDir, server.Client(), server.URL)

	const concurrentRequests = 5
	paths := make([]string, concurrentRequests)
	errCh := make(chan error, concurrentRequests)
	var wg sync.WaitGroup

	for i := range concurrentRequests {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			path, err := service.GetCachedImage(context.Background(), "w500", "poster.jpg")
			paths[idx] = path
			errCh <- err
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("expected concurrent cache fill to succeed, got error: %v", err)
		}
	}

	if requestCount.Load() != 1 {
		t.Fatalf("expected 1 upstream request, got %d", requestCount.Load())
	}

	for i := 1; i < len(paths); i++ {
		if paths[i] != paths[0] {
			t.Fatalf("expected all cache paths to match, got %q and %q", paths[0], paths[i])
		}
	}
	assertTMDBImageContent(t, paths[0], []byte("shared image"))
}

func TestTMDBImageService_CleanupExpiredCache_RemovesExpiredFilesAndKeepsFreshOnes(t *testing.T) {
	cacheDir := t.TempDir()
	service := newTestTMDBImageService(cacheDir, nil, "http://example.com")

	expiredCachePath := filepath.Join(cacheDir, "w500", "expired.jpg")
	freshCachePath := filepath.Join(cacheDir, "w185", "fresh.jpg")
	writeCachedTMDBImage(t, expiredCachePath, []byte("expired image"), time.Now().Add(-2*time.Hour))
	writeCachedTMDBImage(t, freshCachePath, []byte("fresh image"), time.Now())

	if err := service.CleanupExpiredCache(); err != nil {
		t.Fatalf("expected cleanup to succeed, got error: %v", err)
	}

	assertTMDBImageMissing(t, expiredCachePath)
	assertTMDBImageContent(t, freshCachePath, []byte("fresh image"))
	assertTMDBImageMissing(t, filepath.Dir(expiredCachePath))
}

func TestTMDBImageService_CleanupExpiredCache_MissingCacheDir(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "missing-cache")
	service := newTestTMDBImageService(cacheDir, nil, "http://example.com")

	if err := service.CleanupExpiredCache(); err != nil {
		t.Fatalf("expected cleanup with missing cache dir to succeed, got error: %v", err)
	}
}

func writeCachedTMDBImage(t *testing.T, cachePath string, content []byte, modTime time.Time) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(cachePath), 0o750); err != nil {
		t.Fatalf("failed to create cache directory: %v", err)
	}
	if err := os.WriteFile(cachePath, content, 0o600); err != nil {
		t.Fatalf("failed to write cached image: %v", err)
	}
	if err := os.Chtimes(cachePath, modTime, modTime); err != nil {
		t.Fatalf("failed to set cache file times: %v", err)
	}
}

func assertTMDBImageContent(t *testing.T, cachePath string, expected []byte) {
	t.Helper()

	content, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("failed to read cache file: %v", err)
	}
	if string(content) != string(expected) {
		t.Fatalf("expected cache content %q, got %q", string(expected), string(content))
	}
}

func assertTMDBImageMissing(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %q to be removed, got error %v", path, err)
	}
}

func newTestTMDBImageService(cacheDir string, client *http.Client, baseURL string) *TMDBImageService {
	service := NewTMDBImageService(cacheDir, time.Hour, client)
	service.baseURL = baseURL
	return service
}
