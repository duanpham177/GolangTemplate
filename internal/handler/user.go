package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"servergo/internal/service"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, h.svc.List())
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	u, err := h.svc.Create(req.Name)
	if errors.Is(err, service.ErrInvalidName) {
		JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	JSON(w, http.StatusCreated, u)
}

func Health(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
