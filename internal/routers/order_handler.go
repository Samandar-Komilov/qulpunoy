package routers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Samandar-Komilov/qulpunoy/internal/auth"
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
	userID, ok := auth.GetCurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	orders, err := h.service.List(r.Context(), userID)
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
	userID, ok := auth.GetCurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid order id")
		return
	}

	order, err := h.service.GetByID(r.Context(), id, userID)
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

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("Idempotency-Key")

	userID, ok := auth.GetCurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req orderCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	inputs := make([]models.OrderItemInput, 0, len(req.Items))
	for _, i := range req.Items {
		productIDstr, err := uuid.Parse(i.ProductID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid product id")
			return
		}
		inputs = append(inputs, models.OrderItemInput{
			ProductID: productIDstr,
			Quantity:  i.Quantity,
		})
	}

	order, created, err := h.service.Create(r.Context(), userID, key, inputs)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrMissingIdempotencyKey):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, models.ErrEmptyOrderItems),
			errors.Is(err, models.ErrInvalidOrderItemQuantity),
			errors.Is(err, models.ErrDuplicateOrderItem),
			errors.Is(err, models.ErrMissingProductID):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, models.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, models.ErrInsufficientStock):
			writeError(w, http.StatusConflict, err.Error())
		default:
			slog.Error("Failed to create order", "error", err)
			writeError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	writeJSON(w, status, orderResponse{
		ID:          order.ID.String(),
		UserID:      order.UserID.String(),
		Status:      order.Status,
		TotalAmount: order.TotalAmount.String(),
		CreatedAt:   order.CreatedAt.Format(time.RFC3339),
	})
}

func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetCurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid order id")
		return
	}
	order, err := h.service.Cancel(r.Context(), id, userID)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrOrderNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, models.ErrInvalidOrderStatus):
			writeError(w, http.StatusConflict, err.Error())
		default:
			slog.Error("Failed to cancel order", "error", err)
			writeError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, orderResponse{
		ID:          order.ID.String(),
		UserID:      order.UserID.String(),
		Status:      order.Status,
		TotalAmount: order.TotalAmount.String(),
		CreatedAt:   order.CreatedAt.Format(time.RFC3339),
	})
}
