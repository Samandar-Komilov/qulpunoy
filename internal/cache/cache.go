package cache

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const OrderListCacheTTL = 5 * time.Minute
const OrderDetailCacheTTL = 15 * time.Minute

type Cache interface {
	GetJSON(ctx context.Context, key string, dst any) error
	SetJSON(ctx context.Context, key string, val any, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Close() error
}

func OrderListCacheKey(userID uuid.UUID) string {
	return "orders:list:" + userID.String()
}

func OrderDetailCacheKey(id uuid.UUID, userID uuid.UUID) string {
	return "orders:detail:" + id.String() + ":" + userID.String()
}
