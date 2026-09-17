package routers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Samandar-Komilov/qulpunoy/internal/models"
	"github.com/Samandar-Komilov/qulpunoy/internal/services"
)

type AuthHandler struct {
	service services.AuthService
}

func NewAuthHandler(service services.AuthService) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

func (h *AuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	access, refresh, err := h.service.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, models.ErrUnauthorized), errors.Is(err, models.ErrUserNotFound):
			writeError(w, http.StatusUnauthorized, "Invalid credentials")
		case errors.Is(err, models.ErrForbidden):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			slog.Error("Failed to login", "error", err)
			writeError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	access, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, models.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "Invalid refresh token")
			return
		}
		slog.Error("Failed to refresh token", "error", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, refreshResponse{
		AccessToken: access,
	})
}
