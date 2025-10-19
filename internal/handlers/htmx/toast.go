package htmx

import (
	"gowatch/internal/ui/templui/toast"
	"net/http"
)

func (h *Handlers) renderErrorToast(w http.ResponseWriter, r *http.Request, title, description string, duration int) {
	if duration == 0 {
		duration = 4000
	}

	toast.Toast(toast.Props{
		ID:            "toast",
		Title:         title,
		Description:   description,
		Variant:       toast.VariantError,
		Position:      toast.PositionBottomCenter,
		Duration:      duration,
		ShowIndicator: true,
		Icon:          true,
	}).Render(r.Context(), w)
}

func (h *Handlers) renderSuccessToast(w http.ResponseWriter, r *http.Request, title, description string, duration int) {
	if duration == 0 {
		duration = 2000
	}

	toast.Toast(toast.Props{
		ID:            "toast",
		Title:         title,
		Description:   description,
		Variant:       toast.VariantSuccess,
		Position:      toast.PositionBottomCenter,
		Duration:      duration,
		ShowIndicator: true,
		Icon:          true,
	}).Render(r.Context(), w)
}

func (h *Handlers) renderWarningToast(w http.ResponseWriter, r *http.Request, title, description string, duration int) {
	if duration == 0 {
		duration = 3000
	}

	toast.Toast(toast.Props{
		ID:            "toast",
		Title:         title,
		Description:   description,
		Variant:       toast.VariantWarning,
		Position:      toast.PositionBottomCenter,
		Duration:      duration,
		ShowIndicator: true,
		Icon:          true,
	}).Render(r.Context(), w)
}
