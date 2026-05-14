package health

import (
	"encoding/json"
	"net/http"
)

type healthResponse struct {
	Status string `json:"status"`
}

type apiResponse struct {
	Data  any  `json:"data"`
	Error any  `json:"error"`
}

// Handler returns a http.HandlerFunc that responds to liveness checks.
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := apiResponse{
			Data:  healthResponse{Status: "ok"},
			Error: nil,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			// If we can't write the response there is nothing meaningful to do.
			return
		}
	}
}
