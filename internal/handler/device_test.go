package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"servergo/internal/repository"
	"servergo/internal/service"
)

func TestListTelemetryRequiresBearerToken(t *testing.T) {
	svc := service.NewDeviceService(repository.NewTelemetryMemory(), nil)
	h := NewDeviceHandler(svc, "test-api-token")

	request := httptest.NewRequest(http.MethodGet, "/api/v1/devices/sensor-1/telemetry", nil)
	request.SetPathValue("deviceID", "sensor-1")
	recorder := httptest.NewRecorder()
	h.ListTelemetry(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/devices/sensor-1/telemetry", nil)
	request.SetPathValue("deviceID", "sensor-1")
	request.Header.Set("Authorization", "Bearer test-api-token")
	recorder = httptest.NewRecorder()
	h.ListTelemetry(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}
