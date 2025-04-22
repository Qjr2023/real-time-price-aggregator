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
	Set(key string, data *types.PriceData, tierType string) error
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
func (c *RedisCache) Set(key string, data *types.PriceData, tierType string) error {
	ctx := context.Background()
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Determine TTL based on tier type
	var ttl time.Duration
	switch tierType {
	case "hot":
		ttl = 10 * time.Second // hot assets short TTL
	case "medium":
		ttl = 1 * time.Minute // midium assets medium TTL
	case "cold":
		ttl = 5 * time.Minute // cold assets long TTL
	default:
		ttl = 5 * time.Minute
	}

	return c.client.Set(ctx, key, dataBytes, ttl).Err()
}
