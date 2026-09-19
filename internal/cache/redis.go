package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Samandar-Komilov/qulpunoy/internal/models"
	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	client *redis.Client
}

func NewRedisCache(ctx context.Context, addr, password string, db int) (*redisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &redisCache{client: client}, nil
}

func (c *redisCache) GetJSON(ctx context.Context, key string, dst any) error {
	b, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return models.ErrCacheMiss
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

func (c *redisCache) SetJSON(ctx context.Context, key string, val any, ttl time.Duration) error {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, b, ttl).Err()
}

func (c *redisCache) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

func (c *redisCache) Close() error { return c.client.Close() }
