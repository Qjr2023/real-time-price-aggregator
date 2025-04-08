package cache

import (
	"context"
	"encoding/json"
	"time"

	"real-time-price-aggregator/internal/types"

	"github.com/go-redis/redis/v8"
)

// Cache interface defines caching operations
type Cache interface {
	Get(key string) (*types.PriceData, error)
	Set(key string, data *types.PriceData) error
}

// RedisCache implements the Cache interface using Redis
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

// Get retrieves price data from Redis
func (c *RedisCache) Get(key string) (*types.PriceData, error) {
	ctx := context.Background()
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var priceData types.PriceData
	if err := json.Unmarshal(data, &priceData); err != nil {
		return nil, err
	}
	return &priceData, nil
}

// Set stores price data in Redis with a TTL
func (c *RedisCache) Set(key string, data *types.PriceData) error {
	ctx := context.Background()
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, dataBytes, 5*time.Minute).Err()
}
