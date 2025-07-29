package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func jsonResponse(w http.ResponseWriter, status int, body any) {
	log.Debug("Preparing JSON response", "status", status, "body_type", fmt.Sprintf("%T", body))
	w.Header().Set("Content-Type", "application/json")

	// Encode to bytes first to check for errors before writing headers
	data, err := json.Marshal(body)
	if err != nil {
		log.Error("Failed to marshal JSON response", "error", err)
		// If encoding fails, send an error response instead
		http.Error(w, "Failed to encode response as JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	_, writeErr := w.Write(data)
	if writeErr != nil {
		log.Error("Failed to write JSON response", "error", writeErr)
	}
}
