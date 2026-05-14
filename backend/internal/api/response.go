package api

import (
	"encoding/json"
	"net/http"
)

// WriteJSON encodes payload as JSON and writes it with the given HTTP status.
func WriteJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload) //nolint:errcheck
}

// WriteError writes a standard BilimBaga error envelope:
//
//	{ "data": null, "error": { "code": "...", "message": "..." } }
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, map[string]interface{}{
		"data":  nil,
		"error": map[string]string{"code": code, "message": message},
	})
}
