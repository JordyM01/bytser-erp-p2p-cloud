package health

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ReadinessCheck is a function that returns an error if the component
// is not ready to serve traffic.
type ReadinessCheck func() error

// HandleHealthz returns 200 OK if the process is alive.
func HandleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleReadyz returns an http.HandlerFunc that runs all readiness checks.
// If all checks pass, it returns 200. Otherwise it returns 503 with details.
func HandleReadyz(checks ...ReadinessCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		failures := make(map[string]string)
		for i, check := range checks {
			if err := check(); err != nil {
				failures[fmt.Sprintf("check_%d", i)] = err.Error()
			}
		}

		if len(failures) > 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "not_ready",
				"checks": failures,
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	}
}
