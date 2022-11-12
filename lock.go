package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type Locker interface {
	TryLock(ctx context.Context, key string) (bool, error)
	Unlock(ctx context.Context, key string) error
}

type redisLocker struct {
	client     *redis.Client
	prefix     string
	expiration time.Duration
}

func NewRedisLocker(redisClient *redis.Client, prefix string, lockExpiration time.Duration) Locker {
	return &redisLocker{redisClient, prefix, lockExpiration}
}

func (l *redisLocker) TryLock(ctx context.Context, key string) (bool, error) {
	result, err := l.client.SetNX(ctx, l.redisKeyName(key), time.Now().Unix()+int64(l.expiration.Seconds())+1, l.expiration).Result()
	if err != nil {
		return false, fmt.Errorf("redis error: %w", err)
	}
	return result, nil
}

func (l *redisLocker) Unlock(ctx context.Context, key string) error {
	if err := l.client.Del(ctx, l.redisKeyName(key)).Err(); err != nil {
		return fmt.Errorf("redis error: %w", err)
	}
	return nil
}

func (l *redisLocker) redisKeyName(key string) string {
	return fmt.Sprintf("%s:%s", l.prefix, key)
}
