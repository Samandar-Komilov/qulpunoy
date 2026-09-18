package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	OrderStatusPending   = "pending"
	OrderStatusConfirmed = "confirmed"
	OrderStatusCancelled = "cancelled"
)

type Order struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Status      string
	TotalAmount decimal.Decimal // NUMERIC(12,2)
	Items       []OrderItem
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
type OrderItem struct {
	ID            uuid.UUID
	OrderID       uuid.UUID
	ProductID     uuid.UUID
	ProductName   string
	ProductPrice  decimal.Decimal
	Quantity      int
	PriceSnapshot decimal.Decimal
	CreatedAt     time.Time
}

type OrderItemInput struct {
	ProductID uuid.UUID
	Quantity  int
}

type OrderItemLockSelected struct {
	Price decimal.Decimal
	Stock int
}
