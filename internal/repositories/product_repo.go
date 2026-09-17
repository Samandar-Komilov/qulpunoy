package repositories

import (
	"context"
	"fmt"

	"github.com/Samandar-Komilov/qulpunoy/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductRepository struct {
	pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{
		pool: pool,
	}
}

func (r *ProductRepository) Create(ctx context.Context, p *models.Product) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO products (id, name, price, stock_quantity)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, price, stock_quantity, created_at
	`, p.ID, p.Name, p.Price, p.StockQuantity).Scan(&p.ID, &p.Name, &p.Price, &p.StockQuantity, &p.CreatedAt)

	if err != nil {
		return fmt.Errorf("Product create error: %w", err)
	}

	return nil
}
