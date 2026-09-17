package services

import (
	"context"

	"github.com/Samandar-Komilov/qulpunoy/internal/models"
	"github.com/Samandar-Komilov/qulpunoy/internal/repositories"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ProductService interface {
	Create(ctx context.Context, name string, price decimal.Decimal, stock int) (*models.Product, error)
}

type productService struct {
	repo *repositories.ProductRepository
}

func NewProductService(repo *repositories.ProductRepository) *productService {
	return &productService{
		repo: repo,
	}
}

func (s *productService) Create(ctx context.Context, name string, price decimal.Decimal, stock int) (*models.Product, error) {
	if len(name) <= 0 {
		return nil, models.ErrEmptyProductName
	}
	if stock < 0 {
		return nil, models.ErrInvalidStock
	}
	if price.LessThan(decimal.Zero) {
		return nil, models.ErrInvalidPrice
	}

	p := &models.Product{
		ID:            uuid.New(),
		Name:          name,
		Price:         price,
		StockQuantity: stock,
	}
	err := s.repo.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	return p, nil
}
