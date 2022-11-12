package cache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func setupStorage(t *testing.T) CacheStorage {
	testRedisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: testRedisServer.Addr()})

	return NewRedisCacheStorage(redisClient, "storage-test")
}

func TestSettingAndRetrievingValueFromStorage(t *testing.T) {
	storage := setupStorage(t)

	key := "test"
	value := "test value"
	nonExistingKey := "does-not-exist"
	ctx := context.Background()

	err := storage.Set(ctx, key, value)
	if err != nil {
		t.Fatal(err)
	}

	result, err := storage.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if result != value {
		t.Errorf("expected '%s', got '%s'", value, result)
	}

	_, err = storage.Get(ctx, nonExistingKey)
	if err != ErrDoesNotExist {
		t.Errorf("expected '%v', got '%v'", ErrDoesNotExist, err)
	}
}

func TestDeletingValue(t *testing.T) {
	storage := setupStorage(t)

	key := "test"
	value := "test value"
	ctx := context.Background()

	err := storage.Set(ctx, key, value)
	if err != nil {
		t.Fatal(err)
	}

	err = storage.Delete(ctx, key)
	if err != nil {
		t.Fatal(err)
	}

	_, err = storage.Get(ctx, key)
	if err != ErrDoesNotExist {
		t.Errorf("expected '%v', got '%v'", ErrDoesNotExist, err)
	}
}
