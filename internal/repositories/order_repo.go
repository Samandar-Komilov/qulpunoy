package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/Samandar-Komilov/qulpunoy/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{
		pool: pool,
	}
}

func (r *OrderRepository) List(ctx context.Context) ([]models.Order, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, status, total_amount, created_at, updated_at
		FROM orders ORDER BY created_at DESC;
	`)
	if err != nil {
		return nil, fmt.Errorf("Could not read orders from DB: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		o := models.Order{}
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("Could not serialize orders: %w", err)
		}
		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Could not serialize from DB: %w", err)
	}

	return orders, nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	o := &models.Order{}
	items := []models.OrderItem{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, status, total_amount, created_at, updated_at 
		FROM orders WHERE id = $1;
	`, id).Scan(&o.ID, &o.UserID, &o.Status, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrOrderNotFound
		}
		return nil, fmt.Errorf("Failed to get order by id: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT oi.id, p.id, p.name, p.price, oi.quantity, oi.price_snapshot, oi.created_at
		FROM order_items oi
		INNER JOIN products p ON p.id = oi.product_id
		WHERE oi.order_id = $1
		ORDER BY oi.created_at ASC;
	`, id)
	if err != nil {
		return nil, fmt.Errorf("Could not read order items from DB: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		i := models.OrderItem{}
		if err := rows.Scan(&i.ID, &i.ProductID, &i.ProductName, &i.ProductPrice, &i.Quantity, &i.PriceSnapshot, &i.CreatedAt); err != nil {
			return nil, fmt.Errorf("Could not serialize order items: %w", err)
		}
		items = append(items, i)
	}

	o.Items = items

	return o, nil
}
