package models

import "strings"

type PersonRoleDefinition struct {
	Key            string
	Label          string
	MinimumRatings int64
	WatchRole      TopCrewRole
}

var PersonRoles = []PersonRoleDefinition{
	{Key: "director", Label: "Director", MinimumRatings: 2, WatchRole: TopCrewRoleDirector},
	{Key: "writer", Label: "Writer", MinimumRatings: 2, WatchRole: TopCrewRoleWriter},
	{Key: "composer", Label: "Composer", MinimumRatings: 2, WatchRole: TopCrewRoleComposer},
	{Key: "cinematographer", Label: "Cinematographer", MinimumRatings: 2, WatchRole: TopCrewRoleCinematographer},
	{Key: "producer", Label: "Producer", MinimumRatings: 2},
}

func PersonRoleDefinitionByKey(key string) (PersonRoleDefinition, bool) {
	for _, role := range PersonRoles {
		if role.Key == key {
			return role, true
		}
	}
	return PersonRoleDefinition{}, false
}

func PersonFilmographyCrewRoleKey(job string) string {
	job = strings.ToLower(strings.TrimSpace(job))
	switch {
	case strings.Contains(job, "director") && !strings.Contains(job, "photography"):
		return "director"
	case strings.Contains(job, "writer"), strings.Contains(job, "screenplay"), strings.Contains(job, "story"), strings.Contains(job, "novel"), strings.Contains(job, "character"):
		return "writer"
	case strings.Contains(job, "composer"), strings.Contains(job, "music"):
		return "composer"
	case strings.Contains(job, "cinematography"), strings.Contains(job, "photography"):
		return "cinematographer"
	case strings.Contains(job, "producer"):
		return "producer"
	default:
		return "crew"
	}
}
