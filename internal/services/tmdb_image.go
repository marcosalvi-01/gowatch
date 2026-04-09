package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/marcosalvi-01/gowatch/logging"
	"golang.org/x/sync/singleflight"
)

const (
	defaultTMDBImageBaseURL      = "https://image.tmdb.org/t/p"
	defaultTMDBImageClientTimout = 30 * time.Second
)

var (
	ErrInvalidTMDBImageSize = errors.New("invalid TMDB image size")
	ErrInvalidTMDBImagePath = errors.New("invalid TMDB image path")
	ErrTMDBImageNotFound    = errors.New("TMDB image not found")
	ErrTMDBImageUnavailable = errors.New("TMDB image unavailable")

	allowedTMDBImageSizes = map[string]struct{}{
		"w92":   {},
		"w185":  {},
		"w342":  {},
		"w500":  {},
		"w780":  {},
		"w1280": {},
	}
)

type TMDBImageService struct {
	cacheDir string
	cacheTTL time.Duration
	client   *http.Client
	baseURL  string
	group    singleflight.Group
	log      *slog.Logger
}

func NewTMDBImageService(cacheDir string, cacheTTL time.Duration, client *http.Client) *TMDBImageService {
	if client == nil {
		client = &http.Client{Timeout: defaultTMDBImageClientTimout}
	}

	return &TMDBImageService{
		cacheDir: cacheDir,
		cacheTTL: cacheTTL,
		client:   client,
		baseURL:  strings.TrimRight(defaultTMDBImageBaseURL, "/"),
		log:      logging.Get("tmdb image service"),
	}
}

func (s *TMDBImageService) CacheTTL() time.Duration {
	return s.cacheTTL
}

func (s *TMDBImageService) CleanupExpiredCache() error {
	root, err := os.OpenRoot(s.cacheDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("open TMDB image cache root %s: %w", s.cacheDir, err)
	}
	defer func() {
		if closeErr := root.Close(); closeErr != nil {
			s.log.Error("failed to close TMDB image cache root", "path", s.cacheDir, "error", closeErr)
		}
	}()

	deletedCount := 0
	err = filepath.WalkDir(s.cacheDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk TMDB image cache path %s: %w", path, err)
		}
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("read TMDB image cache file info %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if time.Since(info.ModTime()) <= s.cacheTTL {
			return nil
		}

		relPath, err := filepath.Rel(s.cacheDir, path)
		if err != nil {
			return fmt.Errorf("resolve relative TMDB image cache path %s: %w", path, err)
		}

		if err := root.Remove(relPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}

			return fmt.Errorf("remove expired TMDB image cache file %s: %w", path, err)
		}

		deletedCount++

		if err := removeEmptyTMDBImageCacheDirs(root, filepath.Dir(relPath)); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	if deletedCount > 0 {
		s.log.Info("cleaned expired TMDB image cache files", "deletedCount", deletedCount)
	} else {
		s.log.Debug("TMDB image cache cleanup complete", "deletedCount", deletedCount)
	}

	return nil
}

func (s *TMDBImageService) GetCachedImage(ctx context.Context, size, imagePath string) (string, error) {
	// validate image size
	normalizedSize := strings.TrimSpace(size)
	if _, ok := allowedTMDBImageSizes[normalizedSize]; !ok {
		return "", fmt.Errorf("%w: %s", ErrInvalidTMDBImageSize, size)
	}

	normalizedPath, err := normalizeTMDBImagePath(imagePath)
	if err != nil {
		return "", err
	}

	cachePath := filepath.Join(s.cacheDir, size, imagePath)
	isFresh, err := isTMDBImageCacheFresh(cachePath, s.cacheTTL)
	if err == nil && isFresh {
		s.log.Debug("TMDB image cache hit", "size", normalizedSize, "imagePath", normalizedPath)
		return cachePath, nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat TMDB image cache file %s: %w", cachePath, err)
	}

	s.log.Debug("TMDB image cache miss", "size", normalizedSize, "imagePath", normalizedPath)

	result, err, _ := s.group.Do(cachePath, func() (any, error) {
		fresh, freshErr := isTMDBImageCacheFresh(cachePath, s.cacheTTL)
		if freshErr == nil && fresh {
			return cachePath, nil
		}
		if freshErr != nil && !errors.Is(freshErr, os.ErrNotExist) {
			return "", fmt.Errorf("stat TMDB image cache file %s: %w", cachePath, freshErr)
		}

		return s.refreshCachedImage(ctx, normalizedSize, normalizedPath, cachePath)
	})
	if err != nil {
		return "", err
	}

	resolvedPath, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected TMDB image cache result type %T", result)
	}

	return resolvedPath, nil
}

func (s *TMDBImageService) refreshCachedImage(ctx context.Context, size, imagePath, cachePath string) (string, error) {
	// check if cache exists
	_, err := os.Stat(cachePath)
	cacheExists := err == nil

	err = s.downloadImage(context.WithoutCancel(ctx), size, imagePath, cachePath)
	if err == nil {
		s.log.Debug("TMDB image cached", "size", size, "imagePath", imagePath)
		return cachePath, nil
	}

	if cacheExists {
		s.log.Warn("serving stale cached TMDB image after upstream fetch failed", "size", size, "imagePath", imagePath, "error", err)
		return cachePath, nil
	}

	return "", err
}

func (s *TMDBImageService) downloadImage(ctx context.Context, size, imagePath, cachePath string) error {
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o750); err != nil {
		return fmt.Errorf("create TMDB image cache directory %s: %w", filepath.Dir(cachePath), err)
	}

	imageURL := s.baseURL + "/" + size + "/" + imagePath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return fmt.Errorf("create TMDB image request for %s/%s: %w", size, imagePath, err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: fetch %s/%s: %w", ErrTMDBImageUnavailable, size, imagePath, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			s.log.Error("failed to close TMDB image response body", "size", size, "imagePath", imagePath, "error", closeErr)
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s/%s", ErrTMDBImageNotFound, size, imagePath)
	default:
		return fmt.Errorf("%w: fetch %s/%s returned status %d", ErrTMDBImageUnavailable, size, imagePath, resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(cachePath), "tmdb-image-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp TMDB image cache file for %s: %w", cachePath, err)
	}
	tmpPath := tmpFile.Name()

	defer func() {
		if removeErr := os.Remove(tmpPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			s.log.Error("failed to remove temp TMDB image cache file", "path", tmpPath, "error", removeErr)
		}
	}()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write TMDB image cache file %s: %w", tmpPath, err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp TMDB image cache file %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, cachePath); err != nil {
		return fmt.Errorf("replace TMDB image cache file %s: %w", cachePath, err)
	}

	return nil
}

func normalizeTMDBImagePath(imagePath string) (string, error) {
	normalizedPath := strings.TrimSpace(strings.TrimPrefix(imagePath, "/"))
	if normalizedPath == "" {
		return "", fmt.Errorf("%w: image path is empty", ErrInvalidTMDBImagePath)
	}
	if normalizedPath == "." || normalizedPath == ".." {
		return "", fmt.Errorf("%w: %s", ErrInvalidTMDBImagePath, imagePath)
	}
	if strings.ContainsAny(normalizedPath, "/\\") {
		return "", fmt.Errorf("%w: %s", ErrInvalidTMDBImagePath, imagePath)
	}
	if strings.ContainsAny(normalizedPath, "?#") {
		return "", fmt.Errorf("%w: %s", ErrInvalidTMDBImagePath, imagePath)
	}

	return normalizedPath, nil
}

func isTMDBImageCacheFresh(cachePath string, cacheTTL time.Duration) (bool, error) {
	info, err := os.Stat(cachePath)
	if err != nil {
		return false, err
	}

	return time.Since(info.ModTime()) <= cacheTTL, nil
}

func removeEmptyTMDBImageCacheDirs(root *os.Root, dir string) error {
	for {
		cleanDir := filepath.Clean(dir)
		if cleanDir == "." {
			return nil
		}

		entries, err := fs.ReadDir(root.FS(), cleanDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				dir = filepath.Dir(cleanDir)
				continue
			}

			return fmt.Errorf("read TMDB image cache directory %s: %w", cleanDir, err)
		}
		if len(entries) > 0 {
			return nil
		}

		if err := root.Remove(cleanDir); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				dir = filepath.Dir(cleanDir)
				continue
			}

			return fmt.Errorf("remove empty TMDB image cache directory %s: %w", cleanDir, err)
		}

		dir = filepath.Dir(cleanDir)
	}
}
