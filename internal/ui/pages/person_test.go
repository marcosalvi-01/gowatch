package pages

import (
	"net/url"
	"testing"
	"time"

	"github.com/marcosalvi-01/gowatch/internal/models"
)

func TestParsePersonFilmographyState(t *testing.T) {
	tests := []struct {
		name   string
		values url.Values
		want   PersonFilmographyState
	}{
		{
			name: "valid values",
			values: url.Values{
				"credit_type": {"crew"}, "role": {"director"}, "media": {"movie"},
				"quality": {"all"}, "sort": {"oldest"}, "page": {"2"},
			},
			want: PersonFilmographyState{Type: personFilmographyTypeCrew, Role: "director", Media: personMovieMediaType, Quality: personFilmographyQualityAll, Sort: personFilmographySortOldest, Page: 2},
		},
		{
			name:   "missing and invalid values",
			values: url.Values{"sort": {"invalid"}, "page": {"-1"}},
			want:   PersonFilmographyState{Type: personFilmographyTypeAll, Role: personFilmographyRoleAll, Media: personFilmographyMediaAll, Quality: personFilmographyQualityPopular, Sort: personFilmographySortFeatured},
		},
		{
			name:   "page capped",
			values: url.Values{"page": {"999"}},
			want:   PersonFilmographyState{Type: personFilmographyTypeAll, Role: personFilmographyRoleAll, Media: personFilmographyMediaAll, Quality: personFilmographyQualityPopular, Sort: personFilmographySortFeatured, Page: personFilmographyMaxCredits / personFilmographyPageSize},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParsePersonFilmographyState(tt.values); got != tt.want {
				t.Fatalf("state = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestPersonFilmographyCrewRoleKey(t *testing.T) {
	tests := []struct {
		job  string
		want string
	}{
		{job: "Director", want: "director"},
		{job: "Screenplay", want: "writer"},
		{job: "Original Music Composer", want: "composer"},
		{job: "Director of Photography", want: "cinematographer"},
		{job: "Producer", want: "producer"},
		{job: "Best Boy", want: "crew"},
		{job: "", want: "crew"},
	}
	for _, tt := range tests {
		t.Run(tt.job, func(t *testing.T) {
			if got := personFilmographyCrewRoleKey(tt.job); got != tt.want {
				t.Fatalf("role key = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFilterPersonFilmographyByQuality(t *testing.T) {
	credits := buildPersonFilmography([]models.PersonCredit{
		{ID: 1, MediaType: personMovieMediaType, Title: "Popular", VoteCount: 50},
		{ID: 2, MediaType: personMovieMediaType, Title: "Obscure", VoteCount: 49},
	}, nil)

	popular := filterPersonFilmography(credits, PersonFilmographyState{
		Type:    personFilmographyTypeAll,
		Role:    personFilmographyRoleAll,
		Media:   personFilmographyMediaAll,
		Quality: personFilmographyQualityPopular,
	})
	if len(popular) != 1 || popular[0].Credit.Title != "Popular" {
		t.Fatalf("unexpected popular credits: %v", filmographyTitles(popular))
	}

	all := filterPersonFilmography(credits, PersonFilmographyState{
		Type:    personFilmographyTypeAll,
		Role:    personFilmographyRoleAll,
		Media:   personFilmographyMediaAll,
		Quality: personFilmographyQualityAll,
	})
	if len(all) != 2 {
		t.Fatalf("expected all credits, got %d", len(all))
	}
}

func TestBuildPersonFilmographyMergesRoles(t *testing.T) {
	acting := []models.PersonCredit{{
		ID:          10,
		MediaType:   personMovieMediaType,
		Title:       "Shared Movie",
		Role:        "Lead",
		ReleaseDate: personTestDate(2020),
	}}
	crew := []models.PersonCredit{{
		ID:          10,
		MediaType:   personMovieMediaType,
		Title:       "Shared Movie",
		Role:        "Director",
		ReleaseDate: personTestDate(2020),
	}}

	credits := buildPersonFilmography(acting, crew)
	if len(credits) != 1 {
		t.Fatalf("expected merged credit, got %d", len(credits))
	}
	if !credits[0].IsActing || !credits[0].IsCrew {
		t.Fatalf("expected acting and crew flags: %+v", credits[0])
	}
	if !containsString(credits[0].RoleKeys, "actor") || !containsString(credits[0].RoleKeys, "director") {
		t.Fatalf("expected actor and director roles: %+v", credits[0].RoleKeys)
	}
}

func TestSortPersonFilmographyByDate(t *testing.T) {
	credits := buildPersonFilmography(
		[]models.PersonCredit{
			{ID: 1, MediaType: personMovieMediaType, Title: "Older", ReleaseDate: personTestDate(2010)},
			{ID: 2, MediaType: personMovieMediaType, Title: "Newer", ReleaseDate: personTestDate(2020)},
			{ID: 3, MediaType: personMovieMediaType, Title: "Unknown"},
		},
		nil,
	)

	newest := sortPersonFilmography(credits, personFilmographySortNewest)
	if newest[0].Credit.Title != "Newer" || newest[1].Credit.Title != "Older" || newest[2].Credit.Title != "Unknown" {
		t.Fatalf("unexpected newest order: %v", filmographyTitles(newest))
	}

	oldest := sortPersonFilmography(credits, personFilmographySortOldest)
	if oldest[0].Credit.Title != "Older" || oldest[1].Credit.Title != "Newer" || oldest[2].Credit.Title != "Unknown" {
		t.Fatalf("unexpected oldest order: %v", filmographyTitles(oldest))
	}
}

func TestPersonFilmographyPageKeepsTimelineYearsTogether(t *testing.T) {
	credits := make([]models.PersonCredit, 0, 30)
	for i := 0; i < 10; i++ {
		credits = append(credits, models.PersonCredit{
			ID:          int64(i),
			MediaType:   personMovieMediaType,
			Title:       "2020 Movie",
			ReleaseDate: personTestDate(2020),
		})
	}
	for i := 10; i < 20; i++ {
		credits = append(credits, models.PersonCredit{
			ID:          int64(i),
			MediaType:   personMovieMediaType,
			Title:       "2010 Movie",
			ReleaseDate: personTestDate(2010),
		})
	}
	for i := 20; i < 30; i++ {
		credits = append(credits, models.PersonCredit{
			ID:          int64(i),
			MediaType:   personMovieMediaType,
			Title:       "2000 Movie",
			ReleaseDate: personTestDate(2000),
		})
	}

	filmography := buildPersonFilmography(credits, nil)
	page, hasMore, loaded := personFilmographyPage(filmography, personFilmographySortNewest, 0)
	if len(page) != 20 || !hasMore || loaded != 20 {
		t.Fatalf("unexpected first timeline page: len=%d hasMore=%t loaded=%d", len(page), hasMore, loaded)
	}
	for _, credit := range page {
		if credit.Credit.ReleaseDate.Year() == 2000 {
			t.Fatal("first timeline page split year group")
		}
	}

	page, hasMore, loaded = personFilmographyPage(filmography, personFilmographySortNewest, 1)
	if len(page) != 10 || hasMore || loaded != 30 {
		t.Fatalf("unexpected second timeline page: len=%d hasMore=%t loaded=%d", len(page), hasMore, loaded)
	}
}

func TestPersonFilmographyPageChunksGridCredits(t *testing.T) {
	credits := make([]personFilmographyCredit, 50)
	for i := range credits {
		credits[i] = personFilmographyCredit{Credit: models.PersonCredit{ID: int64(i)}}
	}

	first, hasMore, loaded := personFilmographyPage(credits, personFilmographySortRating, 0)
	if len(first) != personFilmographyPageSize || !hasMore || loaded != personFilmographyPageSize {
		t.Fatalf("unexpected first grid page: len=%d hasMore=%t loaded=%d", len(first), hasMore, loaded)
	}

	second, hasMore, loaded := personFilmographyPage(credits, personFilmographySortRating, 1)
	if len(second) != personFilmographyPageSize || !hasMore || loaded != personFilmographyPageSize*2 {
		t.Fatalf("unexpected second grid page: len=%d hasMore=%t loaded=%d", len(second), hasMore, loaded)
	}

	last, hasMore, loaded := personFilmographyPage(credits, personFilmographySortRating, 2)
	if len(last) != 2 || hasMore || loaded != len(credits) {
		t.Fatalf("unexpected last grid page: len=%d hasMore=%t loaded=%d", len(last), hasMore, loaded)
	}
}

func personTestDate(year int) *time.Time {
	date := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	return &date
}

func filmographyTitles(credits []personFilmographyCredit) []string {
	titles := make([]string, len(credits))
	for i, credit := range credits {
		titles[i] = credit.Credit.Title
	}
	return titles
}
