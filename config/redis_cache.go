package config

import (
	"context"
	"time"

	"github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

type RedisCache[T any] struct {
	cache *cache.Cache
	ttl   time.Duration
	group singleflight.Group
}

// NewRedisCacheOf initializes the RedisCache with a TTL, and local TinyLFU cache.
func NewRedisCacheOf[T any](client *redis.Client, ttl time.Duration) *RedisCache[T] {
	localCache := cache.New(&cache.Options{
		Redis:      client,
		LocalCache: cache.NewTinyLFU(1000, ttl),
	})

	return &RedisCache[T]{
		cache: localCache,
		ttl:   ttl,
	}
}

// Get retrieves a value from the cache or loads it using the load function if not found.
func (c *RedisCache[T]) Get(ctx context.Context, key []byte, load func(ctx context.Context) (T, error)) (T, error) {
	var zero T

	keyStr := string(key)

	var result T
	if err := c.cache.Get(ctx, keyStr, &result); err == nil {
		return result, nil
	}

	// Use singleflight to prevent redundant fetches
	v, err, _ := c.group.Do(keyStr, func() (any, error) {
		freshData, loadErr := load(ctx)
		if loadErr != nil {
			return nil, loadErr
		}

		err := c.cache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   keyStr,
			Value: freshData,
			TTL:   c.ttl,
		})
		if err != nil {
			return nil, err
		}

		return freshData, nil
	})

	if err != nil {
		return zero, err
	}

	return v.(T), nil
}
