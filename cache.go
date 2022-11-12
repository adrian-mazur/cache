package cache

import (
	"context"
	"fmt"
	"time"
)

type StringSerializable[V any] interface {
	SerializeToString() (string, error)
	DeserializeFromString(string) (V, error)
}

type Cache[V StringSerializable[V]] struct {
	storage CacheStorage
	locker  Locker
}

func NewCache[V StringSerializable[V]](storage CacheStorage, locker Locker) *Cache[V] {
	return &Cache[V]{storage, locker}
}

// GetOrSetIfDoesNotExist returns the value associated with the key.
// If there is no value associated with the key, valueFunc will be called and the key will be set to its return value.
func (c *Cache[V]) GetOrSetIfDoesNotExist(ctx context.Context, key string, maxRetries int, retryWaitDuration time.Duration, valueFunc func() (V, error)) (V, error) {
	str, err := c.storage.Get(ctx, key)
	var emptyValue V
	if err != nil && err != ErrDoesNotExist {
		return emptyValue, err
	}
	if err == nil {
		return emptyValue.DeserializeFromString(str)
	}
	// key does not exist
	lockObtained, err := c.locker.TryLock(ctx, key)
	if err != nil {
		return emptyValue, err
	}
	if lockObtained {
		defer c.locker.Unlock(ctx, key)
		// it is possible that during the time between checking the value in storage and obtaining the lock, another client has already set the value
		// so it needs to be checked again
		str, err := c.storage.Get(ctx, key)
		if err != nil && err != ErrDoesNotExist {
			return emptyValue, err
		}
		if err == nil {
			return emptyValue.DeserializeFromString(str)
		}
		return c.set(ctx, key, valueFunc)
	}
	// the lock couldn't have been obtained, therefore another client is updating the value concurrently
	// we should be able to get the value after waiting a while
	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return emptyValue, fmt.Errorf("context cancelled")
		case <-time.After(retryWaitDuration):
		}
		str, err := c.storage.Get(ctx, key)
		if err == ErrDoesNotExist {
			continue
		}
		if err != nil {
			return emptyValue, err
		}
		return emptyValue.DeserializeFromString(str)
	}
	return emptyValue, fmt.Errorf("the value couldn't have been retrieved in a reasonable time")
}

// Delete invalidates the cache entry associated with the key.
func (c *Cache[V]) Delete(ctx context.Context, key string) error {
	return c.storage.Delete(ctx, key)
}

func (c *Cache[V]) set(ctx context.Context, key string, valueFunc func() (V, error)) (V, error) {
	var emptyValue V
	value, err := valueFunc()
	if err != nil {
		return emptyValue, err
	}
	str, err := value.SerializeToString()
	if err != nil {
		return emptyValue, fmt.Errorf("serialization error: %w", err)
	}
	if err := c.storage.Set(ctx, key, str); err != nil {
		return emptyValue, err
	}
	return value, nil
}
