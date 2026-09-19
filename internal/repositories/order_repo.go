package repositories

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Samandar-Komilov/qulpunoy/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Could not serialize from DB: %w", err)
	}

	o.Items = items

	return o, nil
}

func (r *OrderRepository) Create(ctx context.Context, userID uuid.UUID, idempotency_key string, items []models.OrderItemInput) (*models.Order, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("Could not begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	orderDefaultStatus := models.OrderStatusPending
	var oid string
	const q_insert_order = `
		INSERT INTO orders (id, user_id, status, total_amount, idempotency_key)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, idempotency_key) DO NOTHING
		RETURNING id
	`
	err = tx.QueryRow(
		ctx, q_insert_order, uuid.New(), userID, orderDefaultStatus, decimal.Zero, idempotency_key,
	).Scan(&oid)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			tx.Rollback(ctx)
			existing, err := r.getByIdempotencyKey(ctx, userID, idempotency_key)
			if err != nil {
				return nil, false, err
			}
			return existing, false, nil
		}
		return nil, false, fmt.Errorf("Could not create an order: %w", err)
	}

	slices.SortFunc(items, func(a, b models.OrderItemInput) int {
		return bytes.Compare(a.ProductID[:], b.ProductID[:])
	})

	sortedIDs := make([]uuid.UUID, 0, len(items))
	for _, i := range items {
		sortedIDs = append(sortedIDs, i.ProductID)
	}

	const q_lock_read = `
		SELECT id, price, stock_quantity FROM products
		WHERE id=ANY($1)
		ORDER BY id ASC
		FOR UPDATE`

	rows, err := tx.Query(ctx, q_lock_read, sortedIDs)
	if err != nil {
		return nil, false, fmt.Errorf("Could not read products from DB: %w", err)
	}
	defer rows.Close()

	products := make(map[uuid.UUID]models.OrderItemLockSelected, len(items))
	for rows.Next() {
		p := models.Product{}
		if err := rows.Scan(&p.ID, &p.Price, &p.StockQuantity); err != nil {
			return nil, false, fmt.Errorf("Could not serialize products: %w", err)
		}
		products[p.ID] = models.OrderItemLockSelected{
			Price: p.Price,
			Stock: p.StockQuantity,
		}
	}

	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("Could not serialize from DB: %w", err)
	}

	total := decimal.Zero
	for _, i := range items {
		p, ok := products[i.ProductID]
		if !ok {
			return nil, false, fmt.Errorf("product %s: %w", i.ProductID, models.ErrProductNotFound)
		}
		if p.Stock < i.Quantity {
			return nil, false, fmt.Errorf("product %s (have %d, want %d): %w", i.ProductID, p.Stock, i.Quantity, models.ErrInsufficientStock)
		}
		total = total.Add(p.Price.Mul(decimal.NewFromInt32(int32(i.Quantity))))
	}

	const q_insert_order_items = `
		INSERT INTO order_items (id, order_id, product_id, quantity, price_snapshot)
		VALUES ($1, $2, $3, $4, $5)
	`
	const q_decrement_stock = `
		UPDATE products
		SET stock_quantity = stock_quantity - $1, updated_at = NOW()
		WHERE id = $2
	`

	for _, it := range items {
		p := products[it.ProductID]
		if _, err := tx.Exec(ctx, q_insert_order_items, uuid.New(), oid, it.ProductID, it.Quantity, p.Price); err != nil {
			return nil, false, fmt.Errorf("Could not insert order item: %w", err)
		}
		if _, err := tx.Exec(ctx, q_decrement_stock, it.Quantity, it.ProductID); err != nil {
			return nil, false, fmt.Errorf("Could not decrement stock: %w", err)
		}
	}

	const q_update_order_total = `
		UPDATE orders SET total_amount = $1, updated_at = NOW() WHERE id = $2
		RETURNING id, user_id, status, total_amount, created_at, updated_at
	`

	var o models.Order
	err = tx.QueryRow(ctx, q_update_order_total, total, oid).Scan(&o.ID, &o.UserID, &o.Status, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, false, fmt.Errorf("Could not update order total: %w", err)
	}

	oitems := []models.OrderItem{}
	const q_select_order_items = `
		SELECT id, order_id, product_id, quantity, price_snapshot
		FROM order_items
		WHERE order_id = $1
	`

	rows, err = tx.Query(ctx, q_select_order_items, o.ID)
	if err != nil {
		return nil, false, fmt.Errorf("Could not select order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		i := models.OrderItem{}
		if err := rows.Scan(&i.ID, &i.OrderID, &i.ProductID, &i.Quantity, &i.PriceSnapshot); err != nil {
			return nil, false, fmt.Errorf("Could not serialize order items: %w", err)
		}
		oitems = append(oitems, i)
	}

	o.Items = oitems

	err = tx.Commit(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("Could not commit an order transaction: %w", err)
	}

	return &o, true, nil
}

func (r *OrderRepository) Cancel(ctx context.Context, id, userID uuid.UUID) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("Could not begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const q_lock_check = `
		SELECT status FROM orders
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`

	var status string
	err = tx.QueryRow(ctx, q_lock_check, id, userID).Scan(
		&status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, models.ErrOrderNotFound
		}
		return false, fmt.Errorf("Could not check order status: %w", err)
	}
	if status != models.OrderStatusPending {
		return false, models.ErrInvalidOrderStatus
	}

	if _, err := tx.Exec(ctx, `UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2`, models.OrderStatusCancelled, id); err != nil {
		return false, fmt.Errorf("Could not cancel order: %w", err)
	}

	const q_select_items = `
		SELECT product_id, quantity FROM order_items WHERE order_id = $1 ORDER BY product_id ASC
	`

	rows, err := tx.Query(ctx, q_select_items, id)
	if err != nil {
		return false, fmt.Errorf("Could not read order items from DB: %w", err)
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		item := models.OrderItem{}
		if err := rows.Scan(&item.ProductID, &item.Quantity); err != nil {
			return false, fmt.Errorf("Could not serialize order items: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("Could not serialize from DB: %w", err)
	}

	const q_restore_stock = `UPDATE products SET stock_quantity = stock_quantity + $1, updated_at = NOW() WHERE id = $2`

	for _, i := range items {
		if _, err := tx.Exec(ctx, q_restore_stock, i.Quantity, i.ProductID); err != nil {
			return false, fmt.Errorf("Could not restore stock: %w", err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return false, fmt.Errorf("Could not commit an order cancel transaction: %w", err)
	}

	return true, nil
}

func (r *OrderRepository) getByIdempotencyKey(ctx context.Context, userID uuid.UUID, idempotency_key string) (*models.Order, error) {
	o := &models.Order{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, status, total_amount, created_at, updated_at 
		FROM orders WHERE user_id = $1 AND idempotency_key = $2;
	`, userID, idempotency_key).Scan(&o.ID, &o.UserID, &o.Status, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrOrderNotFound
		}
		return nil, fmt.Errorf("Failed to get order by idempotency key: %w", err)
	}

	return o, nil
}

func (r *OrderRepository) ExpirePendingOrders(ctx context.Context) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("Could not begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	total_ids := 0
	for {
		const q_select_expired_orders = `
			SELECT id FROM orders
			WHERE status = $1 AND created_at < NOW() - INTERVAL '15 minutes'
			ORDER BY id
			LIMIT 100
			FOR UPDATE SKIP LOCKED
		`
		rows, err := tx.Query(ctx, q_select_expired_orders, models.OrderStatusPending)
		if err != nil {
			return 0, fmt.Errorf("Failed to select expired orders: %w", err)
		}
		var ids []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				return 0, fmt.Errorf("Failed to serialize expired orders: %w", err)
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			return 0, fmt.Errorf("Failed to serialize expired orders: %w", err)
		}
		rows.Close()

		if len(ids) == 0 {
			break
		}

		const q_update_status = `UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2`
		for _, id := range ids {
			if _, err := tx.Exec(ctx, q_update_status, models.OrderStatusCancelled, id); err != nil {
				return 0, fmt.Errorf("Failed to update order status: %w", err)
			}

			const q_select_order_items = `
				SELECT product_id, quantity FROM order_items 
				WHERE order_id = $1 ORDER BY product_id ASC
			`
			itemRows, err := tx.Query(ctx, q_select_order_items, id)
			if err != nil {
				return 0, err
			}
			type restored_item struct {
				pid uuid.UUID
				qty int
			}
			var items_to_restore []restored_item
			for itemRows.Next() {
				var t restored_item
				if err := itemRows.Scan(&t.pid, &t.qty); err != nil {
					return 0, err
				}
				items_to_restore = append(items_to_restore, restored_item{t.pid, t.qty})
			}
			if err := itemRows.Err(); err != nil {
				return 0, err
			}
			itemRows.Close()

			const q_restore_products_stock = `
				UPDATE products SET stock_quantity = stock_quantity + $1, updated_at = NOW() WHERE id = $2
			`
			for _, ri := range items_to_restore {
				if _, err := tx.Exec(ctx, q_restore_products_stock, ri.qty, ri.pid); err != nil {
					return 0, err
				}
			}

		}
		total_ids += len(ids)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return 0, fmt.Errorf("Could not commit an order expire transaction: %w", err)
	}

	return total_ids, nil
}
