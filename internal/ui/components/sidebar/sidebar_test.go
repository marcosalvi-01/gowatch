package sidebar

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/marcosalvi-01/gowatch/internal/models"
)

func TestListsCollapsibleState(t *testing.T) {
	for _, open := range []bool{false, true} {
		var out bytes.Buffer
		if err := listsSection(nil, "", open).Render(context.Background(), &out); err != nil {
			t.Fatal(err)
		}
		html := out.String()
		for _, marker := range []string{`data-sidebar-lists-collapsible="true"`, `data-sidebar-lists-collapsible-trigger="true"`, `aria-expanded="` + map[bool]string{true: "true", false: "false"}[open] + `"`} {
			if !strings.Contains(html, marker) {
				t.Errorf("open=%t: missing %s", open, marker)
			}
		}
	}
}

func TestSidebarIconCollapse(t *testing.T) {
	var out bytes.Buffer
	if err := Sidebar(Props{Collapsed: true, User: &models.User{Name: "User"}}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`data-state="collapsed"`, `data-collapsible="icon"`, `data-templ-collapsible="icon"`, `data-sidebar-lists-collapsible-trigger="true"`} {
		if !strings.Contains(out.String(), marker) {
			t.Errorf("sidebar missing %s", marker)
		}
	}
}
