package services

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Samandar-Komilov/qulpunoy/internal/cache"
	"github.com/Samandar-Komilov/qulpunoy/internal/models"
	"github.com/Samandar-Komilov/qulpunoy/internal/repositories"
	"github.com/google/uuid"
)

type OrderService interface {
	List(ctx context.Context, userID uuid.UUID) ([]models.Order, error)
	GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Order, error)
	Create(ctx context.Context, userID uuid.UUID, idempotency_key string, items []models.OrderItemInput) (*models.Order, bool, error)
	Confirm(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Order, error)
	Cancel(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Order, error)
}

type orderService struct {
	repo  *repositories.OrderRepository
	cache cache.Cache
}

func NewOrderService(repo *repositories.OrderRepository, c cache.Cache) *orderService {
	return &orderService{
		repo:  repo,
		cache: c,
	}
}

func (s *orderService) List(ctx context.Context, userID uuid.UUID) ([]models.Order, error) {
	key := cache.OrderListCacheKey(userID)

	var cached []models.Order
	if err := s.cache.GetJSON(ctx, key, &cached); err == nil {
		return cached, nil
	} else if !errors.Is(err, models.ErrCacheMiss) {
		slog.Debug("Cache get failed, falling through to DB", "key", key, "error", err)
	}

	orders, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.cache.SetJSON(ctx, key, orders, cache.OrderListCacheTTL); err != nil {
		slog.Debug("Cache set failed", "key", key, "error", err)
	}
	return orders, nil
}

func (s *orderService) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Order, error) {
	key := cache.OrderDetailCacheKey(id, userID)

	var cached models.Order
	if err := s.cache.GetJSON(ctx, key, &cached); err == nil {
		return &cached, nil
	} else if !errors.Is(err, models.ErrCacheMiss) {
		slog.Debug("Cache get failed, falling through to DB", "key", key, "error", err)
	}

	order, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if err := s.cache.SetJSON(ctx, key, order, cache.OrderDetailCacheTTL); err != nil {
		slog.Debug("Cache set failed", "key", key, "error", err)
	}
	return order, nil
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

	order, isCreated, err := s.repo.Create(ctx, userID, idempotency_key, items)
	if err != nil {
		return nil, false, err
	}
	if err := s.cache.Del(ctx, cache.OrderListCacheKey(userID)); err != nil {
		slog.Debug("Cache delete failed", "error", err)
	}
	return order, isCreated, nil
}

func (s *orderService) Confirm(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Order, error) {
	isConfirmed, err := s.repo.Confirm(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if !isConfirmed {
		return nil, models.ErrOrderAlreadyConfirmed
	}

	if err := s.cache.Del(ctx, cache.OrderListCacheKey(userID), cache.OrderDetailCacheKey(id, userID)); err != nil {
		slog.Debug("Cache delete failed", "error", err)
	}

	return s.GetByID(ctx, id, userID)
}

func (s *orderService) Cancel(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Order, error) {
	isCancelled, err := s.repo.Cancel(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if !isCancelled {
		return nil, models.ErrOrderNotFound
	}

	if err := s.cache.Del(ctx, cache.OrderListCacheKey(userID), cache.OrderDetailCacheKey(id, userID)); err != nil {
		slog.Debug("Cache delete failed", "error", err)
	}

	return s.GetByID(ctx, id, userID)
}
