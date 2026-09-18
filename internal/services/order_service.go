package services

import (
	"context"

	"github.com/Samandar-Komilov/qulpunoy/internal/models"
	"github.com/Samandar-Komilov/qulpunoy/internal/repositories"
	"github.com/google/uuid"
)

type OrderService interface {
	List(ctx context.Context) ([]models.Order, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error)
	Create(ctx context.Context, userID uuid.UUID, idempotency_key string, items []models.OrderItemInput) (*models.Order, bool, error)
	Cancel(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Order, error)
}

type orderService struct {
	repo *repositories.OrderRepository
}

func NewOrderService(repo *repositories.OrderRepository) *orderService {
	return &orderService{
		repo: repo,
	}
}

func (s *orderService) List(ctx context.Context) ([]models.Order, error) {
	return s.repo.List(ctx)
}

func (s *orderService) GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *orderService) Create(ctx context.Context, userID uuid.UUID, idempotency_key string, items []models.OrderItemInput) (*models.Order, bool, error) {
	if idempotency_key == "" {
		return nil, false, models.ErrMissingIdempotencyKey
	}
	if len(items) == 0 {
		return nil, false, models.ErrEmptyOrderItems
	}
	seen := make(map[uuid.UUID]bool, len(items))
	for _, item := range items {
		if item.Quantity <= 0 {
			return nil, false, models.ErrInvalidOrderItemQuantity
		}
		if _, is_seen := seen[item.ProductID]; is_seen {
			return nil, false, models.ErrDuplicateOrderItem
		}
		seen[item.ProductID] = true
	}

	return s.repo.Create(ctx, userID, idempotency_key, items)
}

func (s *orderService) Cancel(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Order, error) {
	isCancelled, err := s.repo.Cancel(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if !isCancelled {
		return nil, models.ErrOrderNotFound
	}
	return s.repo.GetByID(ctx, id)
}
