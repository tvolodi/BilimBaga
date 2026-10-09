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

// WriteErrorWithDetails writes the standard error envelope plus a structured
// machine-readable "details" object:
//
//	{ "data": null, "error": { "code": "...", "message": "...", "details": {...} } }
func WriteErrorWithDetails(w http.ResponseWriter, status int, code, message string, details map[string]interface{}) {
	WriteJSON(w, status, map[string]interface{}{
		"data":  nil,
		"error": map[string]interface{}{"code": code, "message": message, "details": details},
	})
}
