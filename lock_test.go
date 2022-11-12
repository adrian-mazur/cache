package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func TestTryLock(t *testing.T) {
	testRedisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: testRedisServer.Addr()})

	locker := NewRedisLocker(redisClient, "locker-test", 10*time.Second)
	key := "test"
	ctx := context.Background()

	result, err := locker.TryLock(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("expected true, got false")
	}

	result, err = locker.TryLock(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("expected false, got true")
	}

	err = locker.Unlock(ctx, key)
	if err != nil {
		t.Fatal(err)
	}

	result, err = locker.TryLock(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("expected true, got false")
	}

	err = locker.Unlock(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
}
