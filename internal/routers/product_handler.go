package routers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Samandar-Komilov/qulpunoy/internal/models"
	"github.com/Samandar-Komilov/qulpunoy/internal/services"
)

type ProductHandler struct {
	service services.ProductService
}

func NewProductHandler(service services.ProductService) *ProductHandler {
	return &ProductHandler{
		service: service,
	}
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req productCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	p, err := h.service.Create(r.Context(), req.Name, req.Price, req.StockQuantity)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrEmptyProductName):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, models.ErrInvalidPrice):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, models.ErrInvalidStock):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			slog.Error("Failed to create product", "error", err)
			writeError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, productResponse{
		ID:            p.ID.String(),
		Name:          p.Name,
		Price:         p.Price.String(),
		StockQuantity: p.StockQuantity,
		CreatedAt:     p.CreatedAt.Format(time.RFC3339),
	})
}
