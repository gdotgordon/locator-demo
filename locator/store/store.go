// Package store defines an interface for storing metrics to a store,
// such as redis, and it also implments the RedisStore, which is the
// required implmementation for the demo.  Note we abstract the store
// into an interface so we can run unit tests without having to actually
// store to redis, among other reasons.
package store

import (
	"time"

	"github.com/gdotgordon/locator-demo/locator/locking"
	"github.com/gdotgordon/locator-demo/locator/types"
	"github.com/go-redis/redis"
)

// Store is the data store abstraction.
type Store interface {
	StoreLatency(d time.Duration) error
	AddSuccess() error
	AddError() error
	Clear() error
	AcquireLock() (*locking.Lock, error)
	Unlock(lock *locking.Lock) error
}

// RedisStore implments the Store interface for the Redis client.
type RedisStore struct {
	cli *redis.Client
}

// NewRedisStore does
func NewRedisStore(cli *redis.Client) Store {
	return &RedisStore{cli: cli}
}

// AcquireLock does
func (rs *RedisStore) AcquireLock() (*locking.Lock, error) {
	lck := locking.New(rs.cli, 1*time.Minute, 10)
	return lck, lck.Lock()
}

func (rs *RedisStore) Unlock(lock *locking.Lock) error {
	return lock.Unlock()
}

func (rs *RedisStore) Clear() error {
	return rs.cli.FlushDB().Err()
}

func (rs *RedisStore) StoreLatency(d time.Duration) error {
	return rs.cli.LPush(types.LatencyKey, int64(d)).Err()
}

func (rs *RedisStore) AddSuccess() error {
	return rs.cli.Incr(types.SuccessKey).Err()
}

func (rs *RedisStore) AddError() error {
	return rs.cli.Incr(types.ErrorKey).Err()
}
