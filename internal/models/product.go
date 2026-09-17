package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Product struct {
	ID            uuid.UUID
	Name          string
	Price         decimal.Decimal
	StockQuantity int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
