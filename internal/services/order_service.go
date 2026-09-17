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
