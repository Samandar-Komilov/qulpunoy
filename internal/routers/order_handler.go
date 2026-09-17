package routers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Samandar-Komilov/qulpunoy/internal/models"
	"github.com/Samandar-Komilov/qulpunoy/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type OrderHandler struct {
	service services.OrderService
}

func NewOrderHandler(service services.OrderService) *OrderHandler {
	return &OrderHandler{
		service: service,
	}
}

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	orders, err := h.service.List(r.Context())
	if err != nil {
		slog.Error("Failed to list orders", "error", err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	resp := make([]orderResponse, 0, len(orders))
	for _, o := range orders {
		resp = append(resp, orderResponse{
			ID:          o.ID.String(),
			UserID:      o.UserID.String(),
			Status:      o.Status,
			TotalAmount: o.TotalAmount.String(),
			CreatedAt:   o.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid order id")
		return
	}

	order, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrOrderNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			slog.Error("Failed to get order", "error", err)
			writeError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	var resp orderDetailResponse
	for _, item := range order.Items {
		resp.Items = append(resp.Items, orderItemResponse{
			ID:            item.ID.String(),
			ProductID:     item.ProductID.String(),
			ProductName:   item.ProductName,
			ProductPrice:  item.ProductPrice.String(),
			Quantity:      item.Quantity,
			PriceSnapshot: item.PriceSnapshot.String(),
		})
	}
	resp.orderResponse = orderResponse{
		ID:          order.ID.String(),
		UserID:      order.UserID.String(),
		Status:      order.Status,
		TotalAmount: order.TotalAmount.String(),
		CreatedAt:   order.CreatedAt.Format(time.RFC3339),
	}

	writeJSON(w, http.StatusOK, resp)
}
