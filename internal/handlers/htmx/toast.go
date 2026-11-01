package htmx

import (
	"net/http"

	"github.com/marcosalvi-01/gowatch/internal/ui/shadcn/toast"
)

func RenderErrorToast(w http.ResponseWriter, r *http.Request, title, description string, duration int) {
	renderToast(w, r, title, description, toast.TypeError, duration)
}

func RenderSuccessToast(w http.ResponseWriter, r *http.Request, title, description string, duration int) {
	renderToast(w, r, title, description, toast.TypeSuccess, duration)
}

func RenderWarningToast(w http.ResponseWriter, r *http.Request, title, description string, duration int) {
	renderToast(w, r, title, description, toast.TypeWarning, duration)
}

func RenderInfoToast(w http.ResponseWriter, r *http.Request, title, description string, duration int) {
	renderToast(w, r, title, description, toast.TypeInfo, duration)
}

func renderToast(w http.ResponseWriter, r *http.Request, title, description string, variant toast.Type, duration int) {
	if duration == 0 {
		switch variant {
		case toast.TypeError:
			duration = 4000
		case toast.TypeSuccess:
			duration = 2000
		default:
			duration = 3000
		}
	}

	if err := toast.Toast(toast.Props{
		Title:       title,
		Description: description,
		Type:        variant,
		Timeout:     duration,
	}).Render(r.Context(), w); err != nil {
		log.Error("failed to render toast", "variant", variant, "error", err)
		http.Error(w, "Failed to render "+string(variant)+" notification", http.StatusInternalServerError)
	}
}
