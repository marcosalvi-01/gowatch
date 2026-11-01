package htmx

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderToast(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	RenderErrorToast(w, r, "Error", "Try again", 0)
	for _, want := range []string{`data-templ-toast`, `data-templ-type="error"`, `data-templ-timeout="4000"`, `data-templ-title="Error"`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("toast missing %q", want)
		}
	}
}
