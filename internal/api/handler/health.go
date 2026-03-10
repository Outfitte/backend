package handler

import (
	"encoding/json"
	"net/http"
)

type healthResponse struct {
	Status string `json:"status"`
}

// Health handles GET /health. No auth required.
// r.Context() is extracted here; future handlers pass it to service calls.
func Health(w http.ResponseWriter, r *http.Request) {
	_ = r.Context()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
}
