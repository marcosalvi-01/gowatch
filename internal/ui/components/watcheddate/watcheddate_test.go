package watcheddate

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestPicker(t *testing.T) {
	var out bytes.Buffer
	date := time.Date(2025, time.January, 2, 0, 0, 0, 0, time.UTC)
	if err := Picker("watched-date", date).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`name="watched_date"`, `value="2025-01-02"`, `id="watched-date"`, `data-watched-date="watched-date"`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("date picker missing %q", want)
		}
	}
}
