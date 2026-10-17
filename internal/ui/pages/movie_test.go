package pages

import (
	"testing"

	"github.com/marcosalvi-01/gowatch/internal/models"
)

func TestMovieTrailerLinksOnlyBuildSafeYouTubeTrailerURLs(t *testing.T) {
	videos := []models.RelatedVideo{
		{Key: "abc123", Name: "Official Trailer", Type: "Trailer", Site: "YouTube"},
		{Key: "bad/key", Name: "Injected URL", Type: "Trailer", Site: "YouTube"},
		{Key: "xyz789", Name: "Teaser", Type: "Teaser", Site: "YouTube"},
		{Key: "vimeo123", Name: "Other source", Type: "Trailer", Site: "Vimeo"},
	}

	trailers := movieTrailerLinks(videos)
	if len(trailers) != 1 || trailers[0].URL != "https://www.youtube.com/watch?v=abc123" {
		t.Fatalf("movieTrailerLinks() = %+v, want one safe YouTube trailer", trailers)
	}
	if safeRatingURL("javascript:alert(1)") != "" || safeRatingURL("http://example.com") != "" || movieIMDbURL("") != "" {
		t.Fatal("safeRatingURL() accepted an unsafe URL")
	}
	if got := safeRatingURL("http://www.rottentomatoes.com/m/title"); got != "https://www.rottentomatoes.com/m/title" {
		t.Fatalf("safeRatingURL() did not upgrade Rotten Tomatoes URL: %q", got)
	}
}

func TestRottenTomatoesLogoSelectionUsesScoreThreshold(t *testing.T) {
	tests := []struct {
		name   string
		source string
		score  float64
		want   string
	}{
		{
			name:   "fresh tomato at threshold",
			source: "RT Critics",
			score:  60,
			want:   "/static/images/ratings/tomatometer-fresh.svg",
		},
		{
			name:   "rotten splat below threshold",
			source: "RT Critics",
			score:  59,
			want:   "/static/images/ratings/tomatometer-rotten.svg",
		},
		{
			name:   "hot popcorn at threshold",
			source: "RT Audience",
			score:  60,
			want:   "/static/images/ratings/popcornmeter-hot.svg",
		},
		{
			name:   "stale popcorn below threshold",
			source: "RT Audience",
			score:  59,
			want:   "/static/images/ratings/popcornmeter-stale.svg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, _, _ := ratingSourceAsset(tt.source, tt.score)
			if path != tt.want {
				t.Fatalf("ratingSourceAsset(%q, %v) path = %q, want %q", tt.source, tt.score, path, tt.want)
			}
		})
	}
}
