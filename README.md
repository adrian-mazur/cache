# cache
A cache library using Redis as a storage. This library implements a locking mechanism to ensure that for concurrent requests for the same key, in case if the key is not already present in cache, a function responsible for retreiving the value from database will be called only once.

## Basic usage
```go
redis := redis.NewClient(&redis.Options{})
userCache := cache.NewCache[User](cache.NewRedisCacheStorage(redis, "users"), cache.NewRedisLocker(redis, "users-lock", lockExpiration))
getFromDatabaseFunc := func() (User, error) {
    // function responsible for retrieving the data from a database if the value is not cached
    return user, nil
}
user := userCache.GetOrSetIfDoesNotExist(context, userId, maxRetries, retryWaitDuration, getFromDatabaseFunc)
```
A full working example can be found in the [example/main.go](example/main.go) file.