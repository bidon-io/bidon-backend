package nefta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const StateTTL = 30 * 24 * time.Hour

type State struct {
	NUID            string `json:"nuid"`
	SessionID       int64  `json:"session_id"`
	AdOpportunityID int64  `json:"ad_opportunity_id"`
	LastActivityTS  int64  `json:"last_activity_ts"`
	SessionStartTS  int64  `json:"session_start_ts"`
}

type StateStore interface {
	Find(ctx context.Context, key string) (*State, error)
	Save(ctx context.Context, key string, state *State) error
}

type RedisStateStore struct {
	Redis *redis.ClusterClient
	TTL   time.Duration
}

func NewRedisStateStore(redisClient *redis.ClusterClient) *RedisStateStore {
	return &RedisStateStore{
		Redis: redisClient,
		TTL:   StateTTL,
	}
}

func (s *RedisStateStore) Find(ctx context.Context, key string) (*State, error) {
	value, err := s.Redis.Get(ctx, key).Result()
	switch {
	case errors.Is(err, redis.Nil):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("get nefta state by key: %w", err)
	}

	var state State
	if err = json.Unmarshal([]byte(value), &state); err != nil {
		return nil, fmt.Errorf("unmarshal nefta state: %w", err)
	}

	return &state, nil
}

func (s *RedisStateStore) Save(ctx context.Context, key string, state *State) error {
	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal nefta state: %w", err)
	}

	ttl := s.TTL
	if ttl <= 0 {
		ttl = StateTTL
	}

	if err = s.Redis.Set(ctx, key, string(payload), ttl).Err(); err != nil {
		return fmt.Errorf("save nefta state by key: %w", err)
	}

	return nil
}
