package htmx

import (
	"net/http"

	"gowatch/internal/ui/templui/toast"
)

func (h *Handlers) renderErrorToast(w http.ResponseWriter, r *http.Request, title, description string, duration int) {
	if duration == 0 {
		duration = 4000
	}

	if err := toast.Toast(toast.Props{
		ID:            "toast",
		Title:         title,
		Description:   description,
		Variant:       toast.VariantError,
		Position:      toast.PositionBottomCenter,
		Duration:      duration,
		ShowIndicator: true,
		Icon:          true,
	}).Render(r.Context(), w); err != nil {
		log.Error("failed to render error toast", "error", err)
		http.Error(w, "Failed to render error notification", http.StatusInternalServerError)
	}
}

func (h *Handlers) renderSuccessToast(w http.ResponseWriter, r *http.Request, title, description string, duration int) {
	if duration == 0 {
		duration = 2000
	}

	if err := toast.Toast(toast.Props{
		ID:            "toast",
		Title:         title,
		Description:   description,
		Variant:       toast.VariantSuccess,
		Position:      toast.PositionBottomCenter,
		Duration:      duration,
		ShowIndicator: true,
		Icon:          true,
	}).Render(r.Context(), w); err != nil {
		log.Error("failed to render success toast", "error", err)
		http.Error(w, "Failed to render success notification", http.StatusInternalServerError)
	}
}

func (h *Handlers) renderWarningToast(w http.ResponseWriter, r *http.Request, title, description string, duration int) {
	if duration == 0 {
		duration = 3000
	}

	if err := toast.Toast(toast.Props{
		ID:            "toast",
		Title:         title,
		Description:   description,
		Variant:       toast.VariantWarning,
		Position:      toast.PositionBottomCenter,
		Duration:      duration,
		ShowIndicator: true,
		Icon:          true,
	}).Render(r.Context(), w); err != nil {
		log.Error("failed to render warning toast", "error", err)
		http.Error(w, "Failed to render warning notification", http.StatusInternalServerError)
	}
}
