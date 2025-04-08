package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// Cache interface defines caching operations
type Cache interface {
	Get(symbol string) (CachedPrice, error)
	Set(symbol string, price CachedPrice) error
}

// RedisCache implements the Cache interface
type RedisCache struct {
	client *redis.Client
}

// CachedPrice represents price data stored in cache
type CachedPrice struct {
	Price     float64 `json:"price"`
	Timestamp int64   `json:"timestamp"`
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(client *redis.Client) Cache {
	return &RedisCache{client: client}
}

// Get retrieves price data from Redis
func (c *RedisCache) Get(symbol string) (CachedPrice, error) {
	log.Printf("Fetching %s from Redis cache", symbol)
	data, err := c.client.Get(context.Background(), symbol).Bytes()
	if err == redis.Nil {
		log.Printf("Cache miss for %s", symbol)
		return CachedPrice{}, fmt.Errorf("cache miss")
	}
	if err != nil {
		log.Printf("Redis error for %s: %v", symbol, err)
		return CachedPrice{}, err
	}

	var price CachedPrice
	if err := json.Unmarshal(data, &price); err != nil {
		log.Printf("Failed to unmarshal cached data for %s: %v", symbol, err)
		return CachedPrice{}, err
	}

	log.Printf("Cache hit for %s: price=%f, timestamp=%d",
		symbol, price.Price, price.Timestamp)
	return price, nil
}

// Set saves price data to Redis
func (c *RedisCache) Set(symbol string, price CachedPrice) error {
	data, err := json.Marshal(price)
	if err != nil {
		log.Printf("Failed to marshal price for %s: %v", symbol, err)
		return err
	}

	// Use 5 minute TTL for testing
	ttl := 5 * time.Minute
	log.Printf("Setting cache for %s: price=%f, timestamp=%d, ttl=%v",
		symbol, price.Price, price.Timestamp, ttl)

	err = c.client.Set(context.Background(), symbol, data, ttl).Err()
	if err != nil {
		log.Printf("Redis set error for %s: %v", symbol, err)
		return err
	}

	log.Printf("Successfully cached %s", symbol)
	return nil
}
