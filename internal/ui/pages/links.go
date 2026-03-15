package pages

import "strconv"

func personPagePath(personID int64) string {
	if personID <= 0 {
		return ""
	}

	return "/person/" + strconv.FormatInt(personID, 10)
}
