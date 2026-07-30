package pages

import (
	"net/url"
	"strconv"

	"github.com/marcosalvi-01/gowatch/internal/models"
)

type PersonFilmographyState struct {
	Type    string
	Role    string
	Media   string
	Sort    string
	Quality string
	Page    int
}

func ParsePersonFilmographyState(values url.Values) PersonFilmographyState {
	state := PersonFilmographyState{
		Type:    values.Get("credit_type"),
		Role:    values.Get("role"),
		Media:   values.Get("media"),
		Sort:    values.Get("sort"),
		Quality: values.Get("quality"),
	}

	if state.Type != personFilmographyTypeActing && state.Type != personFilmographyTypeCrew {
		state.Type = personFilmographyTypeAll
	}
	if state.Role == "" {
		state.Role = personFilmographyRoleAll
	}
	if state.Media != personMovieMediaType && state.Media != "tv" {
		state.Media = personFilmographyMediaAll
	}
	if state.Quality != personFilmographyQualityAll {
		state.Quality = personFilmographyQualityPopular
	}
	switch state.Sort {
	case personFilmographySortNewest, personFilmographySortOldest, personFilmographySortRating, personFilmographySortPopular:
	default:
		state.Sort = personFilmographySortFeatured
	}
	if page, err := strconv.Atoi(values.Get("page")); err == nil && page > 0 {
		state.Page = min(page, personFilmographyMaxCredits/personFilmographyPageSize)
	}

	return state
}

type personPreparedFilmography struct {
	State       PersonFilmographyState
	Credits     []personFilmographyCredit
	Roles       []personFilmographyRoleOption
	Visible     []personFilmographyCredit
	Groups      []personFilmographyYearGroup
	HasMore     bool
	LoadedCount int
	Total       int
}

func preparePersonFilmography(actingCredits, crewCredits []models.PersonCredit, state PersonFilmographyState) personPreparedFilmography {
	credits := buildPersonFilmography(actingCredits, crewCredits)
	roles := personFilmographyRoleOptions(credits)
	state.Role = personFilmographyRoleSelection(state.Role, roles)
	filtered := filterPersonFilmography(credits, state)
	sorted := capPersonFilmography(sortPersonFilmography(filtered, state.Sort))
	visible, hasMore, loadedCount := personFilmographyPage(sorted, state.Sort, state.Page)

	prepared := personPreparedFilmography{
		State:       state,
		Credits:     sorted,
		Roles:       roles,
		Visible:     visible,
		HasMore:     hasMore,
		LoadedCount: loadedCount,
		Total:       len(sorted),
	}
	if state.Sort == personFilmographySortNewest || state.Sort == personFilmographySortOldest {
		prepared.Groups = groupPersonFilmographyByYear(visible)
	}
	return prepared
}

func personFilmographyPath(personID int64, state PersonFilmographyState, page int) string {
	query := url.Values{}
	query.Set("credit_type", state.Type)
	query.Set("role", state.Role)
	query.Set("media", state.Media)
	query.Set("sort", state.Sort)
	query.Set("quality", state.Quality)
	query.Set("page", strconv.Itoa(page))
	return personPagePath(personID) + "?" + query.Encode()
}
