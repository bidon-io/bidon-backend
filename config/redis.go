package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient builds a redis.UniversalClient from REDIS_URL. A single address yields a
// standalone *redis.Client, multiple comma-separated addresses yield a *redis.ClusterClient.
func NewRedisClient(poolSize int) (redis.UniversalClient, error) {
	addrsEnv := os.Getenv("REDIS_URL")
	if addrsEnv == "" {
		return nil, fmt.Errorf("REDIS_URL is not set")
	}

	opts := &redis.UniversalOptions{
		PoolSize: poolSize,
	}

	entries := strings.Split(addrsEnv, ",")
	if len(entries) == 1 && strings.Contains(entries[0], "://") {
		parsed, err := redis.ParseURL(entries[0])
		if err != nil {
			return nil, fmt.Errorf("redis.ParseURL(%v): %w", entries[0], err)
		}
		opts.Addrs = []string{parsed.Addr}
		opts.DB = parsed.DB
		opts.Username = parsed.Username
		opts.Password = parsed.Password
		opts.TLSConfig = parsed.TLSConfig
	} else {
		opts.Addrs = entries
	}

	return redis.NewUniversalClient(opts), nil
}

type RedisPinger struct {
	client redis.Cmdable
}

func NewRedisPinger(rdb redis.Cmdable) Pinger {
	return &RedisPinger{
		client: rdb,
	}
}

func (r *RedisPinger) Ping(ctx context.Context) error {
	if r.client == nil {
		return nil
	}

	cmd := r.client.Ping(ctx)
	if err := cmd.Err(); err != nil {
		return err
	}

	return nil
}
