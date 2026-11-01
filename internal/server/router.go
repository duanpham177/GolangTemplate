package server

import (
	"log/slog"
	"net/http"
	"time"

	"servergo/internal/handler"
)

// NewRouter registers all routes (Go 1.22+ method-aware patterns).
func NewRouter(logger *slog.Logger, users *handler.UserHandler, devices *handler.DeviceHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health)
	mux.HandleFunc("GET /api/v1/users", users.List)
	mux.HandleFunc("POST /api/v1/users", users.Create)
	mux.HandleFunc("GET /api/v1/devices/{deviceID}/telemetry", devices.ListTelemetry)
	mux.HandleFunc("POST /api/v1/devices/{deviceID}/commands", devices.SendCommand)
	return logging(logger, mux)
}

func logging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}
