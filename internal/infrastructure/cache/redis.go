package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	appcache "practice-ddd/internal/application/cache"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) appcache.Cache {
	return &RedisCache{client: client}
}

func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	value, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", appcache.ErrMiss
	}
	return value, err
}

func (c *RedisCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}
