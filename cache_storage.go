package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
)

var ErrDoesNotExist = errors.New("item identified by a given key does not exist")

type CacheStorage interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string) error
	Delete(ctx context.Context, key string) error
}

type redisCacheStorage struct {
	client *redis.Client
	prefix string
}

func NewRedisCacheStorage(redisClient *redis.Client, prefix string) CacheStorage {
	return &redisCacheStorage{redisClient, prefix}
}

func (s *redisCacheStorage) Get(ctx context.Context, key string) (string, error) {
	val, err := s.client.Get(ctx, s.redisKeyName(key)).Result()
	if err != nil {
		if err == redis.Nil {
			return "", ErrDoesNotExist
		}
		return "", fmt.Errorf("redis error: %w", err)
	}
	return val, nil
}

func (s *redisCacheStorage) Set(ctx context.Context, key string, value string) error {
	if err := s.client.Set(ctx, s.redisKeyName(key), value, 0).Err(); err != nil {
		return fmt.Errorf("redis error: %w", err)
	}
	return nil
}

func (s *redisCacheStorage) Delete(ctx context.Context, key string) error {
	if err := s.client.Del(ctx, s.redisKeyName(key)).Err(); err != nil {
		return fmt.Errorf("redis error: %w", err)
	}
	return nil
}

func (s *redisCacheStorage) redisKeyName(key string) string {
	return fmt.Sprintf("%s:%s", s.prefix, key)
}
