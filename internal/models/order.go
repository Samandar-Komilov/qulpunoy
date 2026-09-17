package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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
