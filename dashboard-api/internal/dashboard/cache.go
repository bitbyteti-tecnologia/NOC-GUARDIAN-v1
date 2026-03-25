package dashboard

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
}

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(url string) *RedisCache {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil
	}
	return &RedisCache{client: redis.NewClient(opt)}
}

func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *RedisCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func EncodeCache(v AggregateResponse) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DecodeCache(raw string) (AggregateResponse, error) {
	var v AggregateResponse
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return AggregateResponse{}, err
	}
	return v, nil
}
