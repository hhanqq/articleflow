package httptransport

import "net/http"

func NewHealthHandler(serviceName string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte(`{"service":"` + serviceName + `","status":"ok"}`))
	})
}
