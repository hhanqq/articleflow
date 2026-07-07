package httptransport

import (
	"encoding/json"
	"net/http"
	"time"
)

type HealthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Time    string `json:"time"`
}

func NewHealthHandler(serviceName string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(response).Encode(HealthResponse{
			Service: serviceName,
			Status:  "ok",
			Time:    time.Now().UTC().Format(time.RFC3339),
		})
	})
}
