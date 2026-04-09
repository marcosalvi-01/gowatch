package utils

import (
	"net/url"
	"strings"
)

const tmdbImageRoutePrefix = "/images/tmdb/"

func TMDBImageURL(size, imagePath string) string {
	normalizedSize := strings.TrimSpace(size)
	normalizedPath := strings.TrimSpace(strings.TrimPrefix(imagePath, "/"))
	if normalizedSize == "" || normalizedPath == "" {
		return ""
	}
	if strings.ContainsAny(normalizedPath, "/\\") {
		return ""
	}

	return tmdbImageRoutePrefix + normalizedSize + "/" + url.PathEscape(normalizedPath)
}
