package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

type mockCacheItem string

func (i mockCacheItem) SerializeToString() (string, error) {
	return string(i), nil
}
func (i mockCacheItem) DeserializeFromString(str string) (mockCacheItem, error) {
	return mockCacheItem(str), nil
}

func TestGetOrSetIfDoesNotExist(t *testing.T) {
	testRedisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: testRedisServer.Addr()})
	storage := NewRedisCacheStorage(redisClient, "storage-test")
	locker := NewRedisLocker(redisClient, "locker-test", 10*time.Second)

	cache := NewCache[mockCacheItem](storage, locker)

	ctx := context.Background()
	key := "test"
	value := mockCacheItem("test")
	maxRetries := 5
	retryWaitDuration := 100 * time.Millisecond
	goroutinesNum := 10

	var valueFuncCalledTimes int32
	valueFunc := func() (mockCacheItem, error) {
		atomic.AddInt32(&valueFuncCalledTimes, 1)
		return value, nil
	}

	var wg sync.WaitGroup
	wg.Add(goroutinesNum)
	for i := 0; i < goroutinesNum; i++ {
		go func() {
			result, err := cache.GetOrSetIfDoesNotExist(ctx, key, maxRetries, retryWaitDuration, valueFunc)
			if err != nil {
				t.Error(err)
				wg.Done()
				return
			}
			if result != value {
				t.Errorf("expected '%s', got '%s'", value, result)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if valueFuncCalledTimes != 1 {
		t.Errorf("expected 1, got %d", valueFuncCalledTimes)
	}
}
