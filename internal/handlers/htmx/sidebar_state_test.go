package htmx

import "testing"

func TestSidebarPathFromURL(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		output string
	}{
		{name: "full URL", input: "http://localhost:8080/watchlist", output: "/watchlist"},
		{name: "path with query", input: "/list/42?sort=desc", output: "/list/42"},
		{name: "trailing slash", input: "/stats/", output: "/stats"},
		{name: "invalid URL", input: "%", output: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := sidebarPathFromURL(testCase.input)
			if result != testCase.output {
				t.Fatalf("expected %q, got %q", testCase.output, result)
			}
		})
	}
}

func TestSidebarCurrentPageFromPath(t *testing.T) {
	testCases := []struct {
		name   string
		path   string
		output string
	}{
		{name: "home", path: "/home", output: "home"},
		{name: "list page", path: "/list/42", output: "list_42"},
		{name: "non-sidebar page", path: "/movie/123", output: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := sidebarCurrentPageFromPath(testCase.path)
			if result != testCase.output {
				t.Fatalf("expected %q, got %q", testCase.output, result)
			}
		})
	}
}

func TestResolveSidebarCurrentPage(t *testing.T) {
	testCases := []struct {
		name         string
		currentURL   string
		selectedPath string
		output       string
	}{
		{
			name:         "prefer current URL when available",
			currentURL:   "http://localhost:8080/watchlist",
			selectedPath: "/list/3",
			output:       "watchlist",
		},
		{
			name:         "fallback to selected list path",
			currentURL:   "http://localhost:8080/movie/123",
			selectedPath: "/list/3",
			output:       "list_3",
		},
		{
			name:         "empty when both are invalid",
			currentURL:   "http://localhost:8080/movie/123",
			selectedPath: "/search",
			output:       "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := resolveSidebarCurrentPage(testCase.currentURL, testCase.selectedPath)
			if result != testCase.output {
				t.Fatalf("expected %q, got %q", testCase.output, result)
			}
		})
	}
}

func TestResolveSidebarListsOpen(t *testing.T) {
	testCases := []struct {
		name        string
		currentPage string
		cookieValue string
		output      bool
	}{
		{
			name:        "list page defaults open when cookie missing",
			currentPage: "list_12",
			cookieValue: "",
			output:      true,
		},
		{
			name:        "cookie true opens lists",
			currentPage: "watchlist",
			cookieValue: "true",
			output:      true,
		},
		{
			name:        "cookie false has priority over list page",
			currentPage: "list_12",
			cookieValue: "false",
			output:      false,
		},
		{
			name:        "empty cookie defaults closed",
			currentPage: "home",
			cookieValue: "",
			output:      false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := resolveSidebarListsOpen(testCase.currentPage, testCase.cookieValue)
			if result != testCase.output {
				t.Fatalf("expected %t, got %t", testCase.output, result)
			}
		})
	}
}
