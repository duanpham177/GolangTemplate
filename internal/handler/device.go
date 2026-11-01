package handler

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"servergo/internal/service"
)

type DeviceHandler struct {
	svc      *service.DeviceService
	apiToken string
}

func NewDeviceHandler(svc *service.DeviceService, apiToken string) *DeviceHandler {
	return &DeviceHandler{svc: svc, apiToken: apiToken}
}

func (h *DeviceHandler) ListTelemetry(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r) {
		JSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	records, err := h.svc.ListTelemetry(r.PathValue("deviceID"))
	if errors.Is(err, service.ErrInvalidDeviceID) {
		JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	JSON(w, http.StatusOK, records)
}

func (h *DeviceHandler) SendCommand(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r) {
		JSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req struct {
		Command string `json:"command"`
	}
	if err := decoder.Decode(&req); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "request must contain one json object"})
		return
	}

	err := h.svc.SendCommand(r.PathValue("deviceID"), req.Command)
	switch {
	case errors.Is(err, service.ErrInvalidDeviceID), errors.Is(err, service.ErrInvalidCommand):
		JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, service.ErrDeviceOffline):
		JSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	case err != nil:
		JSON(w, http.StatusInternalServerError, map[string]string{"error": "could not send command"})
	default:
		JSON(w, http.StatusAccepted, map[string]string{"status": "sent"})
	}
}

func (h *DeviceHandler) authorized(r *http.Request) bool {
	const prefix = "Bearer "
	if h.apiToken == "" {
		return false
	}
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	token := strings.TrimPrefix(header, prefix)
	return len(token) == len(h.apiToken) &&
		subtle.ConstantTimeCompare([]byte(token), []byte(h.apiToken)) == 1
}
