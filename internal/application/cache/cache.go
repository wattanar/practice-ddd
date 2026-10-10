package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrMiss = errors.New("cache miss")

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
}

func decode(value string, dst any) error {
	return json.Unmarshal([]byte(value), dst)
}

func encode(value any) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
