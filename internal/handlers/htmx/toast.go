package htmx

import (
	"net/http"

	"github.com/marcosalvi-01/gowatch/internal/ui/templui/toast"
)

const toastID = "toast"

func RenderErrorToast(w http.ResponseWriter, r *http.Request, title, description string, duration int) {
	renderToast(w, r, title, description, toast.VariantError, duration)
}

func RenderSuccessToast(w http.ResponseWriter, r *http.Request, title, description string, duration int) {
	renderToast(w, r, title, description, toast.VariantSuccess, duration)
}

func RenderWarningToast(w http.ResponseWriter, r *http.Request, title, description string, duration int) {
	renderToast(w, r, title, description, toast.VariantWarning, duration)
}

func RenderInfoToast(w http.ResponseWriter, r *http.Request, title, description string, duration int) {
	renderToast(w, r, title, description, toast.VariantInfo, duration)
}

func renderToast(w http.ResponseWriter, r *http.Request, title, description string, variant toast.Variant, duration int) {
	if duration == 0 {
		switch variant {
		case toast.VariantError:
			duration = 4000
		case toast.VariantSuccess:
			duration = 2000
		default:
			duration = 3000
		}
	}

	if err := toast.Toast(toast.Props{
		ID:            toastID,
		Title:         title,
		Description:   description,
		Variant:       variant,
		Position:      toast.PositionBottomCenter,
		Duration:      duration,
		ShowIndicator: true,
		Icon:          true,
	}).Render(r.Context(), w); err != nil {
		log.Error("failed to render toast", "variant", variant, "error", err)
		http.Error(w, "Failed to render "+string(variant)+" notification", http.StatusInternalServerError)
	}
}
